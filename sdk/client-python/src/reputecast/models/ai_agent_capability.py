from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_agent_capability_state import AIAgentCapabilityState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AIAgentCapability")



@_attrs_define
class AIAgentCapability:
    """ 
        Attributes:
            state (AIAgentCapabilityState):
            checked_at (datetime.datetime | Unset):
     """

    state: AIAgentCapabilityState
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        state = self.state.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "state": state,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state = AIAgentCapabilityState(d.pop("state"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        ai_agent_capability = cls(
            state=state,
            checked_at=checked_at,
        )

        return ai_agent_capability

