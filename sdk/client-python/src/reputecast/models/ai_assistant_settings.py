from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.ai_agent_limits import AIAgentLimits





T = TypeVar("T", bound="AIAssistantSettings")



@_attrs_define
class AIAssistantSettings:
    """ 
        Attributes:
            enabled (bool):
            limits (AIAgentLimits):
            per_person (int):
            per_workspace (int):
            revision (int):
     """

    enabled: bool
    limits: AIAgentLimits
    per_person: int
    per_workspace: int
    revision: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_agent_limits import AIAgentLimits # noqa: PLC0415
        enabled = self.enabled

        limits = self.limits.to_dict()

        per_person = self.per_person

        per_workspace = self.per_workspace

        revision = self.revision


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "limits": limits,
            "perPerson": per_person,
            "perWorkspace": per_workspace,
            "revision": revision,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_agent_limits import AIAgentLimits # noqa: PLC0415
        d = dict(src_dict)
        enabled = d.pop("enabled")

        limits = AIAgentLimits.from_dict(d.pop("limits"))




        per_person = d.pop("perPerson")

        per_workspace = d.pop("perWorkspace")

        revision = d.pop("revision")

        ai_assistant_settings = cls(
            enabled=enabled,
            limits=limits,
            per_person=per_person,
            per_workspace=per_workspace,
            revision=revision,
        )

        return ai_assistant_settings

