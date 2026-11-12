from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.review_ab_kind import ReviewABKind






T = TypeVar("T", bound="ReviewAB")



@_attrs_define
class ReviewAB:
    """ 
        Attributes:
            duration_hours (int):
            kind (ReviewABKind):
            label (str):
            test_percent (int):
            variants (int):
     """

    duration_hours: int
    kind: ReviewABKind
    label: str
    test_percent: int
    variants: int





    def to_dict(self) -> dict[str, Any]:
        duration_hours = self.duration_hours

        kind = self.kind.value

        label = self.label

        test_percent = self.test_percent

        variants = self.variants


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "durationHours": duration_hours,
            "kind": kind,
            "label": label,
            "testPercent": test_percent,
            "variants": variants,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        duration_hours = d.pop("durationHours")

        kind = ReviewABKind(d.pop("kind"))




        label = d.pop("label")

        test_percent = d.pop("testPercent")

        variants = d.pop("variants")

        review_ab = cls(
            duration_hours=duration_hours,
            kind=kind,
            label=label,
            test_percent=test_percent,
            variants=variants,
        )

        return review_ab

