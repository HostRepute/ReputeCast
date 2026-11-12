from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderHeldDecisionResult")



@_attrs_define
class ProviderHeldDecisionResult:
    """ 
        Attributes:
            action (str):
            moved (int):
     """

    action: str
    moved: int





    def to_dict(self) -> dict[str, Any]:
        action = self.action

        moved = self.moved


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "moved": moved,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = d.pop("action")

        moved = d.pop("moved")

        provider_held_decision_result = cls(
            action=action,
            moved=moved,
        )

        return provider_held_decision_result

