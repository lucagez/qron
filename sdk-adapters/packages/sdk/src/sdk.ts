import { getSdk, TinyJob } from "./generated"
import { GraphQLClient } from 'graphql-request'
import dayjs from "dayjs"
import type { ManipulateType } from "dayjs"

// TODO: ID are strings. Should modify that
// https://github.com/dotansimha/graphql-code-generator/issues/688#issuecomment-439358697
export const tinyq = (url: string) => {
  const sdk = getSdk(new GraphQLClient(url, {
    headers: {
      'Authorization': 'some-auth-token'
    }
  }))
  return sdk
}

type JobConfig<T> = {
  name: string
  expr: string
  start_at?: Date
  initialState: T
  timeout?: number

  // admin
  _register?: boolean
}

const jobs = []

export type TinyRequest<T> = Omit<TinyJob, 'state'> & { 
  state: T
  
  retry: (state?: T) => Retry<T>
  fail: (state?: T) => Fail<T>
  commit: (state?: T) => Commit<T>

  // TODO: Stop `status` does not exists
  stop: (state?: T) => Stop<T>
}

class TinyResponseBuilder<T> {
  request: TinyRequest<T>

  constructor(request: TinyRequest<T>) {
    this.request = request
  }

  withState(state: T) {
    this.request.state = state
    return this
  }

  // TODO: Should have `dump` method that serialize + encrypt
  dump() {
    return JSON.stringify({
      ...this.request,
      state: JSON.stringify(this.request.state),
    })
  }
}

type TimeUnit = 'mins' | 'hours' | 'days' | 'weeks' | 'months' | 'years'

export class Retry<T> extends TinyResponseBuilder<T> {
  constructor(request: TinyRequest<T>) {
    super(request)

    // Default delay
    this.request.expr = '@after 1 mins'
    this.request.status = 'READY'
  }

  afterInterval(interval: number, unit: TimeUnit) {
    this.request.expr = `@after ${interval} ${unit}`
    return this
  }

  afterMinutes(interval: number) {
    this.request.expr = `@after ${interval} mins`
    return this
  }

  afterHours(interval: number) {
    this.request.expr = `@after ${interval} hours`
    return this
  }

  afterDays(interval: number) {
    this.request.expr = `@after ${interval} days`
    return this
  }

  afterWeeks(interval: number) {
    this.request.expr = `@after ${interval} weeks`
    return this
  }

  afterMonths(interval: number) {
    this.request.expr = `@after ${interval} months`
    return this
  }

  afterYears(interval: number) {
    this.request.expr = `@after ${interval} years`
    return this
  }

  afterTime(time: Date) {
    this.request.expr = `@at ${time.toISOString()}`
    return this
  }
}

export class Commit<T> extends TinyResponseBuilder<T> {
  constructor(request: TinyRequest<T>) {
    super(request)

    this.request.status = 'SUCCESS'
  }
}

export class Stop<T> extends TinyResponseBuilder<T> {
  constructor(request: TinyRequest<T>) {
    super(request)

    this.request.status = 'STOPPED'
  }
}

export class Fail<T> extends TinyResponseBuilder<T> {
  constructor(request: TinyRequest<T>) {
    super(request)

    this.request.status = 'FAILED'
  }
}

export class TinyRequestBuilder<T> {
  request = {} as TinyRequest<T>
  client = tinyq('http://some-addr.com')

  // TODO: serialize + encrypt
  withState(state: T) {
    this.request.state = state
    return this
  }

  withMeta(meta: any) {
    this.request.meta = JSON.stringify(meta)
    return this
  }

  withName(name: string) {
    this.request.name = name
    return this
  }

  startsAt(date: Date) {
    this.request.start_at = date
    return this
  }

  startsAfter(interval: number, unit: TimeUnit) {
    // TODO: Check compatibility
    this.request.start_at = dayjs().add(interval, unit as ManipulateType)
    return this
  }

  async schedule(state?: T) {
    if (state) {
      state = this.withState(state).request.state
    }

    return this.client.createJob({
      args: {
        meta: this.request.meta,
        state: JSON.stringify(this.request.state),
        expr: this.request.expr,
        name: this.request.name || 'some random name if not provided',
        start_at: this.request.start_at,
      }
    }) 
  }
}

export class Cron<T> extends TinyRequestBuilder<T> {
  every(interval: number, unit: TimeUnit) {
    this.request.expr = `@every ${interval} ${unit}`
    return this
  }

  everyMinutes(interval: number) {
    this.request.expr = `@every ${interval} mins`
    return this
  }

  everyHours(interval: number) {
    this.request.expr = `@every ${interval} hours`
    return this
  }

  everyDays(interval: number) {
    this.request.expr = `@every ${interval} days`
    return this
  }

  everyWeeks(interval: number) {
    this.request.expr = `@every ${interval} weeks`
    return this
  }

  everyMonths(interval: number) {
    this.request.expr = `@every ${interval} months`
    return this
  }

  everyYears(interval: number) {
    this.request.expr = `@every ${interval} years`
    return this
  }
}

export class Job<T> extends TinyRequestBuilder<T> {
  after(interval: number, unit: TimeUnit) {
    this.request.expr = `@after ${interval} ${unit}`
    return this
  }

  at(date: Date) {
    this.request.expr = `@at ${date.toISOString()}`
    return this
  }

  afterMinutes(interval: number) {
    this.request.expr = `@after ${interval} mins`
    return this
  }

  afterHours(interval: number) {
    this.request.expr = `@after ${interval} hours`
    return this
  }

  afterDays(interval: number) {
    this.request.expr = `@after ${interval} days`
    return this
  }

  afterWeeks(interval: number) {
    this.request.expr = `@after ${interval} weeks`
    return this
  }

  afterMonths(interval: number) {
    this.request.expr = `@after ${interval} months`
    return this
  }

  afterYears(interval: number) {
    this.request.expr = `@after ${interval} years`
    return this
  }
}

export const createHandler = <T>() =>{
  const requestUtil = (request: TinyRequest<T>) => {
    return {
      ...request,
      retry: (state?: T) => {
        const _retry = new Retry(request)
        return state ? _retry.withState(state) : _retry
      },
      commit: (state?: T) => {
        const _commit = new Commit(request)
        return state ? _commit.withState(state) : _commit
      },
      stop: (state?: T) => {
        const _stop = new Stop(request)
        return state ? _stop.withState(state) : _stop
      },
      fail: (state?: T) => {
        const _fail = new Fail(request)
        return state ? _fail.withState(state) : _fail
      },
    }
  }

  const hydrateRequest = (raw: any) => {
    try {
      // decrypt state + deserialize
      const state = JSON.parse(raw.state)
      return {
        ...raw,
        state,
      }
    } catch (err) {
      // TODO: do something for automatic retry
      console.error(err)
      throw new Error('wrong format for request')
    }
  }

  // TODO: implement dehydrate response with encryption
  // -> move here from class

  return {
    requestUtil,
    hydrateRequest,
    cron: new Cron<T>(),
    job: new Job<T>(),
  }
}
