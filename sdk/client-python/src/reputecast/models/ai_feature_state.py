from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_feature_state_key import AIFeatureStateKey
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="AIFeatureState")



@_attrs_define
class AIFeatureState:
    """ 
        Attributes:
            available (bool):
            key (AIFeatureStateKey):
            on (bool):
            model_id (UUID | Unset):
     """

    available: bool
    key: AIFeatureStateKey
    on: bool
    model_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        key = self.key.value

        on = self.on

        model_id: str | Unset = UNSET
        if not isinstance(self.model_id, Unset):
            model_id = str(self.model_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "key": key,
            "on": on,
        })
        if model_id is not UNSET:
            field_dict["modelId"] = model_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        key = AIFeatureStateKey(d.pop("key"))




        on = d.pop("on")

        _model_id = d.pop("modelId", UNSET)
        model_id: UUID | Unset
        if isinstance(_model_id,  Unset):
            model_id = UNSET
        else:
            model_id = UUID(_model_id)




        ai_feature_state = cls(
            available=available,
            key=key,
            on=on,
            model_id=model_id,
        )

        return ai_feature_state

