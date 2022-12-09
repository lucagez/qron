import { NextApiHandler } from 'next'
import { createHandler, TinyRequest, Retry, Commit, Fail, Stop } from 'sdk'



const tinyNext = <T>(
  x: (state: TinyRequest<T>) => Promise<Retry<T> | Commit<T> | Stop<T> | Fail<T>>
) => {
  const { requestUtil, job, cron } = createHandler<T>()

  const handle: NextApiHandler = async (req, res) => {
    try {
      const tinyResponse = await x(requestUtil({
        state: JSON.parse(req.body),
      } as any) as any)

      // RIPARTIRE QUI!<---
      // - Should be able to serialize/deserialize TinyJob to/from json

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