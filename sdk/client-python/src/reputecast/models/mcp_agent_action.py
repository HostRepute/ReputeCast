from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.mcp_agent_action_result import MCPAgentActionResult
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="MCPAgentAction")



@_attrs_define
class MCPAgentAction:
    """ 
        Attributes:
            agent (str):
            id (str):
            occurred_at (datetime.datetime):
            proposal (bool):
            result (MCPAgentActionResult):
            target (str):
            tool (str):
            agent_id (str | Unset):
            note (str | Unset):
            summary (str | Unset):
     """

    agent: str
    id: str
    occurred_at: datetime.datetime
    proposal: bool
    result: MCPAgentActionResult
    target: str
    tool: str
    agent_id: str | Unset = UNSET
    note: str | Unset = UNSET
    summary: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        agent = self.agent

        id = self.id

        occurred_at = self.occurred_at.isoformat()

        proposal = self.proposal

        result = self.result.value

        target = self.target

        tool = self.tool

        agent_id = self.agent_id

        note = self.note

        summary = self.summary


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "agent": agent,
            "id": id,
            "occurredAt": occurred_at,
            "proposal": proposal,
            "result": result,
            "target": target,
            "tool": tool,
        })
        if agent_id is not UNSET:
            field_dict["agentId"] = agent_id
        if note is not UNSET:
            field_dict["note"] = note
        if summary is not UNSET:
            field_dict["summary"] = summary

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        agent = d.pop("agent")

        id = d.pop("id")

        occurred_at = datetime.datetime.fromisoformat(d.pop("occurredAt"))




        proposal = d.pop("proposal")

        result = MCPAgentActionResult(d.pop("result"))




        target = d.pop("target")

        tool = d.pop("tool")

        agent_id = d.pop("agentId", UNSET)

        note = d.pop("note", UNSET)

        summary = d.pop("summary", UNSET)

        mcp_agent_action = cls(
            agent=agent,
            id=id,
            occurred_at=occurred_at,
            proposal=proposal,
            result=result,
            target=target,
            tool=tool,
            agent_id=agent_id,
            note=note,
            summary=summary,
        )

        return mcp_agent_action

