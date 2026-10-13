import { http } from '@/api/client'
import { setJobsSource, type BackgroundJob, type JobsSource } from '@/lib/jobs'

/**
 * The jobs tray's server list: imports running in the background and those
 * that ended in the last two hours (GET /workspaces/{workspace}/jobs).
 * Exports are synchronous downloads; the screens that start one track it
 * with trackJob() while it runs.
 */
export const importJobsSource: JobsSource = {
  async list(workspace, signal) {
    const { data } = await http.GET('/api/v1/workspaces/{workspace}/jobs', { params: { path: { workspace } }, signal })
    return (data?.items ?? []) as BackgroundJob[]
  },
  async cancel(workspace, id) {
    await http.POST('/api/v1/workspaces/{workspace}/imports/{id}/cancel', { params: { path: { workspace, id } } })
  },
}

setJobsSource(importJobsSource)
