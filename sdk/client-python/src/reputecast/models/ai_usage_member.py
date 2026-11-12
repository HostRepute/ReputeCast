from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AIUsageMember")



@_attrs_define
class AIUsageMember:
    """ 
        Attributes:
            cost_micros (int):
            name (str):
            requests (int):
            tokens (int):
            user_id (str | Unset):
     """

    cost_micros: int
    name: str
    requests: int
    tokens: int
    user_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        cost_micros = self.cost_micros

        name = self.name

        requests = self.requests

        tokens = self.tokens

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "costMicros": cost_micros,
            "name": name,
            "requests": requests,
            "tokens": tokens,
        })
        if user_id is not UNSET:
            field_dict["userId"] = user_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost_micros = d.pop("costMicros")

        name = d.pop("name")

        requests = d.pop("requests")

        tokens = d.pop("tokens")

        user_id = d.pop("userId", UNSET)

        ai_usage_member = cls(
            cost_micros=cost_micros,
            name=name,
            requests=requests,
            tokens=tokens,
            user_id=user_id,
        )

        return ai_usage_member

