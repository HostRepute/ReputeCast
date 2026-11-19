from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AIAgentLimits")



@_attrs_define
class AIAgentLimits:
    """ 
        Attributes:
            cost_micros (int):
            tokens (int):
            tool_calls (int):
     """

    cost_micros: int
    tokens: int
    tool_calls: int





    def to_dict(self) -> dict[str, Any]:
        cost_micros = self.cost_micros

        tokens = self.tokens

        tool_calls = self.tool_calls


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "costMicros": cost_micros,
            "tokens": tokens,
            "toolCalls": tool_calls,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost_micros = d.pop("costMicros")

        tokens = d.pop("tokens")

        tool_calls = d.pop("toolCalls")

        ai_agent_limits = cls(
            cost_micros=cost_micros,
            tokens=tokens,
            tool_calls=tool_calls,
        )

        return ai_agent_limits

