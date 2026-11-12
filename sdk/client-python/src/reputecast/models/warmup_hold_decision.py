from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.warmup_hold_decision_action import WarmupHoldDecisionAction
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="WarmupHoldDecision")



@_attrs_define
class WarmupHoldDecision:
    """ 
        Attributes:
            action (WarmupHoldDecisionAction):
            target_provider_id (UUID | Unset):
     """

    action: WarmupHoldDecisionAction
    target_provider_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        target_provider_id: str | Unset = UNSET
        if not isinstance(self.target_provider_id, Unset):
            target_provider_id = str(self.target_provider_id)


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
        action = WarmupHoldDecisionAction(d.pop("action"))




        _target_provider_id = d.pop("targetProviderId", UNSET)
        target_provider_id: UUID | Unset
        if isinstance(_target_provider_id,  Unset):
            target_provider_id = UNSET
        else:
            target_provider_id = UUID(_target_provider_id)




        warmup_hold_decision = cls(
            action=action,
            target_provider_id=target_provider_id,
        )

        return warmup_hold_decision

