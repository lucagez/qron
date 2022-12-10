import { NextApiHandler } from 'next'
import { createHandler, TinyRequest, Retry, Commit, Fail, Stop } from 'sdk'



const tinyNext = <T>(
  x: (request: TinyRequest<T>) => Promise<Retry<T> | Commit<T> | Stop<T> | Fail<T>>
) => {
  const { requestUtil, hydrateRequest, job, cron } = createHandler<T>()

  const handle: NextApiHandler = async (req, res) => {
    try {
      console.log('received raw request:', req.body)
      const hydrated = hydrateRequest(req.body)
      console.log('received request:', hydrated)
      const tinyResponse = await x(requestUtil(hydrated))

      // RIPARTIRE QUI!<---
      // - test flow

      console.log('sending back:', tinyResponse.dump())
      res.status(200)
      // raw
      res.end(tinyResponse.dump())
    } catch(error) {
      console.error(error)
      res.status(500)
      res.send(error)
    }
  }

  return {
    handle,
    job,
    cron,
  }
}



const { handle, job: counter, cron: counterJob } = tinyNext<{counter: number}>(async ({ state, retry, commit }) => {
  console.log('receiving counter job with:', state, typeof state)
  if (state.counter < 10) {
    console.log('should retry', state.counter)

    return retry({ counter: state.counter + 1})
      .afterHours(1)
  }
  return commit()
})

export default handle