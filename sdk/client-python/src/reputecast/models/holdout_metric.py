from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.holdout_metric_key import HoldoutMetricKey






T = TypeVar("T", bound="HoldoutMetric")



@_attrs_define
class HoldoutMetric:
    """ 
        Attributes:
            confidence (float):
            holdout (float):
            key (HoldoutMetricKey):
            lift (float):
            marketing (float):
            significant (bool):
            tested (bool):
     """

    confidence: float
    holdout: float
    key: HoldoutMetricKey
    lift: float
    marketing: float
    significant: bool
    tested: bool





    def to_dict(self) -> dict[str, Any]:
        confidence = self.confidence

        holdout = self.holdout

        key = self.key.value

        lift = self.lift

        marketing = self.marketing

        significant = self.significant

        tested = self.tested


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confidence": confidence,
            "holdout": holdout,
            "key": key,
            "lift": lift,
            "marketing": marketing,
            "significant": significant,
            "tested": tested,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confidence = d.pop("confidence")

        holdout = d.pop("holdout")

        key = HoldoutMetricKey(d.pop("key"))




        lift = d.pop("lift")

        marketing = d.pop("marketing")

        significant = d.pop("significant")

        tested = d.pop("tested")

        holdout_metric = cls(
            confidence=confidence,
            holdout=holdout,
            key=key,
            lift=lift,
            marketing=marketing,
            significant=significant,
            tested=tested,
        )

        return holdout_metric

