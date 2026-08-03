import {
  ApiClient,
  AuthApi,
  SystemApi,
  WebhooksApi
} from '@/api/generated'

jest.mock('superagent', () => jest.fn())

describe('generated Keel API client', () => {
  test('imports with same-origin defaults and configurable authentication', () => {
    const defaultClient = new ApiClient()
    const customClient = new ApiClient('/keel')

    expect(defaultClient.basePath).toBe('')
    expect(defaultClient.enableCookies).toBe(false)
    expect(customClient.basePath).toBe('/keel')

    customClient.authentications.BasicAuth.username = 'admin'
    customClient.authentications.BasicAuth.password = 'secret'
    customClient.authentications.BearerAuth.apiKey = 'Bearer token'

    expect(customClient.authentications.BasicAuth.username).toBe('admin')
    expect(customClient.authentications.BearerAuth.apiKey).toBe('Bearer token')
  })

  test('uses stable paths, methods, and security declarations', async () => {
    const client = new ApiClient()
    client.callApi = jest.fn(() => Promise.resolve({
      data: { name: 'keel' },
      response: { status: 200 }
    }))

    const system = new SystemApi(client)
    const auth = new AuthApi(client)
    const webhooks = new WebhooksApi(client)

    await expect(system.getVersion()).resolves.toEqual({ name: 'keel' })
    expect(client.callApi.mock.calls[0][0]).toBe('/version')
    expect(client.callApi.mock.calls[0][1]).toBe('GET')
    expect(client.callApi.mock.calls[0][7]).toEqual([])

    await auth.getAuthInfoWithHttpInfo()
    expect(client.callApi.mock.calls[1][0]).toBe('/v1/auth/info')
    expect(client.callApi.mock.calls[1][7]).toEqual(['BasicAuth', 'BearerAuth'])

    await webhooks.receiveRegistryWebhookWithHttpInfo({ events: [] })
    expect(client.callApi.mock.calls[2][0]).toBe('/v1/webhooks/registry')
    expect(client.callApi.mock.calls[2][7]).toEqual([])

    await webhooks.receiveNativeWebhookWithHttpInfo({ name: 'keel', tag: 'latest' })
    expect(client.callApi.mock.calls[3][0]).toBe('/v1/webhooks/native')
    expect(client.callApi.mock.calls[3][7]).toEqual(['BasicAuth', 'BearerAuth'])
  })
})
