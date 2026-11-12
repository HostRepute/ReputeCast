from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.mcp_overview_transport import MCPOverviewTransport
from typing import cast

if TYPE_CHECKING:
  from ..models.mcp_agent_row import MCPAgentRow
  from ..models.mcp_tool_info import MCPToolInfo





T = TypeVar("T", bound="MCPOverview")



@_attrs_define
class MCPOverview:
    """ 
        Attributes:
            agents (list[MCPAgentRow]):
            approval_expiry_hours (int):
            can_manage (bool):
            capabilities (list[str]):
            enabled (bool):
            endpoint (str):
            prompts (list[str]):
            rate_limit_per_minute (int):
            resources (list[str]):
            tools (list[MCPToolInfo]):
            transport (MCPOverviewTransport):
     """

    agents: list[MCPAgentRow]
    approval_expiry_hours: int
    can_manage: bool
    capabilities: list[str]
    enabled: bool
    endpoint: str
    prompts: list[str]
    rate_limit_per_minute: int
    resources: list[str]
    tools: list[MCPToolInfo]
    transport: MCPOverviewTransport





    def to_dict(self) -> dict[str, Any]:
        from ..models.mcp_agent_row import MCPAgentRow # noqa: PLC0415
        from ..models.mcp_tool_info import MCPToolInfo # noqa: PLC0415
        agents = []
        for agents_item_data in self.agents:
            agents_item = agents_item_data.to_dict()
            agents.append(agents_item)



        approval_expiry_hours = self.approval_expiry_hours

        can_manage = self.can_manage

        capabilities = self.capabilities



        enabled = self.enabled

        endpoint = self.endpoint

        prompts = self.prompts



        rate_limit_per_minute = self.rate_limit_per_minute

        resources = self.resources



        tools = []
        for tools_item_data in self.tools:
            tools_item = tools_item_data.to_dict()
            tools.append(tools_item)



        transport = self.transport.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "agents": agents,
            "approvalExpiryHours": approval_expiry_hours,
            "canManage": can_manage,
            "capabilities": capabilities,
            "enabled": enabled,
            "endpoint": endpoint,
            "prompts": prompts,
            "rateLimitPerMinute": rate_limit_per_minute,
            "resources": resources,
            "tools": tools,
            "transport": transport,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mcp_agent_row import MCPAgentRow # noqa: PLC0415
        from ..models.mcp_tool_info import MCPToolInfo # noqa: PLC0415
        d = dict(src_dict)
        agents = []
        _agents = d.pop("agents")
        for agents_item_data in (_agents):
            agents_item = MCPAgentRow.from_dict(agents_item_data)



            agents.append(agents_item)


        approval_expiry_hours = d.pop("approvalExpiryHours")

        can_manage = d.pop("canManage")

        capabilities = cast(list[str], d.pop("capabilities"))


        enabled = d.pop("enabled")

        endpoint = d.pop("endpoint")

        prompts = cast(list[str], d.pop("prompts"))


        rate_limit_per_minute = d.pop("rateLimitPerMinute")

        resources = cast(list[str], d.pop("resources"))


        tools = []
        _tools = d.pop("tools")
        for tools_item_data in (_tools):
            tools_item = MCPToolInfo.from_dict(tools_item_data)



            tools.append(tools_item)


        transport = MCPOverviewTransport(d.pop("transport"))




        mcp_overview = cls(
            agents=agents,
            approval_expiry_hours=approval_expiry_hours,
            can_manage=can_manage,
            capabilities=capabilities,
            enabled=enabled,
            endpoint=endpoint,
            prompts=prompts,
            rate_limit_per_minute=rate_limit_per_minute,
            resources=resources,
            tools=tools,
            transport=transport,
        )

        return mcp_overview

