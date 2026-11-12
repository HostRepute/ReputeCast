from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_tracking_action_input_action import ProviderTrackingActionInputAction






T = TypeVar("T", bound="ProviderTrackingActionInput")



@_attrs_define
class ProviderTrackingActionInput:
    """ 
        Attributes:
            action (ProviderTrackingActionInputAction):
     """

    action: ProviderTrackingActionInputAction





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = ProviderTrackingActionInputAction(d.pop("action"))




        provider_tracking_action_input = cls(
            action=action,
        )

        return provider_tracking_action_input

