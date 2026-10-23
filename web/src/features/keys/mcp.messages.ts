import { defineMessages } from '@/i18n'

/**
 * Settings › Keys & MCP › MCP server (A5 §5.10, SettingsMCP): the endpoint,
 * the agents' capabilities, recent calls, the tools and the limits. Tool
 * names (get_campaign_report), permission keys and the client configuration
 * stay as they are.
 */
export const m = defineMessages('keys.mcp', {
  title: 'Settings',
  serverOn: 'MCP server on',
  serverOff: 'MCP server off',
  addAgent: 'Add agent',
  forbidden: "Your role can't manage MCP agents",
  loadFailed: "Couldn't load the MCP server settings",
  noAgentsTitle: 'No MCP agents',
  noAgentsBody: 'Give an AI assistant read access first; grant drafting or sending later, with approvals.',

  // Connect.
  connectTitle: 'Let AI agents work in this workspace, with limits',
  connectBody:
    'ReputeCast exposes a Model Context Protocol server. Each agent gets its own token and permission set; anything that sends mail needs a human approval unless you say otherwise. Every call lands in the audit log.',
  endpoint: 'Endpoint',
  transport: 'Transport',
  transportHttp: 'Streamable HTTP · OAuth 2.1 or per-agent token',
  transportStdio: 'stdio · local bridge (run an MCP proxy next to the client)',
  transportHint: 'The server speaks Streamable HTTP; stdio clients connect through a local bridge.',
  connectFrom: 'Connect from Claude Desktop, Claude Code or any MCP client',
  clientConfig: 'Client configuration',

  // The agents matrix. The capability columns and cells come from the server, in the app language.
  agentsTitle: 'Agents · {count}',
  agentsCaption: 'Permissions per agent. "Approve" means a team member confirms in ReputeCast before the action runs.',
  agentsTable: 'Agents and what they can do',
  colAgent: 'Agent',
  colLastCall: 'Last call',
  colActions: 'Actions',
  agentLine: 'token {masked} · created by {name} · {permissions}',
  editAgent: 'Edit {name} in Keys & MCP',
  never: 'Never',

  // Recent agent actions.
  actionsTitle: 'Recent agent actions',
  fullAuditLog: 'Full audit log',
  actionsLoadFailed: "Couldn't load the agent actions",
  actionsLoading: 'Loading the agent actions',
  actionsTable: 'Recent agent actions',
  colTime: 'Time',
  colTool: 'Tool',
  colResult: 'Result',
  colApproval: 'Approval',
  noCalls: 'No agent calls yet.',
  notNeeded: 'Not needed',
  notNeededNote: 'Not needed · {note}',
  denied: 'Denied',
  error: 'Error',
  review: 'Review',

  // Tools and limits.
  toolsTitle: 'Tools exposed · {count}',
  toolsList: 'Tools',
  toolTitle: '{description} Needs {permission}.',
  resourcesPrompts: 'Resources: {resources}. Prompts: {prompts}.',
  limitsList: 'Limits',
  limitRate: 'Rate limit each agent to {count, plural, one {{rate} call} other {{rate} calls}} per minute',
  limitApprovals: 'Approvals expire after {hours, plural, one {{n} hour} other {{n} hours}}',
  limitSecrets: "Agents never see API keys, SSO settings or other agents' tokens",
  changeLimits: 'Change limits',
  limitsEyebrow: 'Dialog · MCP server limits',
  limitsTitle: 'MCP server limits',
  limitsDesc: 'Apply to every agent in this workspace. Changes are logged.',
  saveLimits: 'Save limits',
  limitsSaved: 'MCP server limits saved',
  serverOnToggle: 'MCP server on · agents can connect',
  ratePerAgent: 'Calls per minute, per agent',
  approvalHours: 'Approvals expire after (hours)',
  notSaved: 'Not saved',

  loading: 'Loading the MCP server settings',
  loadingAgents: 'Loading the agents',
})
