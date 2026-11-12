from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_held_decision_input_action import ProviderHeldDecisionInputAction
from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderHeldDecisionInput")



@_attrs_define
class ProviderHeldDecisionInput:
    """ 
        Attributes:
            action (ProviderHeldDecisionInputAction):
            target_provider_id (str | Unset): Provider to move the held share to (move)
     """

    action: ProviderHeldDecisionInputAction
    target_provider_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        target_provider_id = self.target_provider_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
        })
        if target_provider_id is not UNSET:
            field_dict["targetProviderId"] = target_provider_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = ProviderHeldDecisionInputAction(d.pop("action"))




        target_provider_id = d.pop("targetProviderId", UNSET)

        provider_held_decision_input = cls(
            action=action,
            target_provider_id=target_provider_id,
        )

        return provider_held_decision_input

