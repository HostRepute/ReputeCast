from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_usage_feature_feature import AIUsageFeatureFeature






T = TypeVar("T", bound="AIUsageFeature")



@_attrs_define
class AIUsageFeature:
    """ 
        Attributes:
            cost_micros (int):
            feature (AIUsageFeatureFeature):
            requests (int):
            tokens (int):
     """

    cost_micros: int
    feature: AIUsageFeatureFeature
    requests: int
    tokens: int





    def to_dict(self) -> dict[str, Any]:
        cost_micros = self.cost_micros

        feature = self.feature.value

        requests = self.requests

        tokens = self.tokens


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "costMicros": cost_micros,
            "feature": feature,
            "requests": requests,
            "tokens": tokens,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost_micros = d.pop("costMicros")

        feature = AIUsageFeatureFeature(d.pop("feature"))




        requests = d.pop("requests")

        tokens = d.pop("tokens")

        ai_usage_feature = cls(
            cost_micros=cost_micros,
            feature=feature,
            requests=requests,
            tokens=tokens,
        )

        return ai_usage_feature

