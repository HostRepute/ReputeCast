from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_feature_default_feature import AIFeatureDefaultFeature






T = TypeVar("T", bound="AIFeatureDefault")



@_attrs_define
class AIFeatureDefault:
    """ 
        Attributes:
            feature (AIFeatureDefaultFeature):
            model_id (str):
     """

    feature: AIFeatureDefaultFeature
    model_id: str





    def to_dict(self) -> dict[str, Any]:
        feature = self.feature.value

        model_id = self.model_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "feature": feature,
            "modelId": model_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        feature = AIFeatureDefaultFeature(d.pop("feature"))




        model_id = d.pop("modelId")

        ai_feature_default = cls(
            feature=feature,
            model_id=model_id,
        )

        return ai_feature_default

