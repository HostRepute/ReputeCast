from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_feature_state_key import AIFeatureStateKey






T = TypeVar("T", bound="AIFeatureState")



@_attrs_define
class AIFeatureState:
    """ 
        Attributes:
            available (bool):
            key (AIFeatureStateKey):
            on (bool):
     """

    available: bool
    key: AIFeatureStateKey
    on: bool





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        key = self.key.value

        on = self.on


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "key": key,
            "on": on,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        key = AIFeatureStateKey(d.pop("key"))




        on = d.pop("on")

        ai_feature_state = cls(
            available=available,
            key=key,
            on=on,
        )

        return ai_feature_state

