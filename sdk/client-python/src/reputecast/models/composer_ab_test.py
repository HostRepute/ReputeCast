from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_ab_test_kind import ComposerABTestKind
from ..models.composer_ab_test_metric import ComposerABTestMetric
from ..models.composer_ab_test_no_winner import ComposerABTestNoWinner
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.composer_ab_variant import ComposerABVariant





T = TypeVar("T", bound="ComposerABTest")



@_attrs_define
class ComposerABTest:
    """ 
        Attributes:
            duration_hours (int):
            kind (ComposerABTestKind):
            metric (ComposerABTestMetric):
            test_percent (int): Share of the audience in the test
            variants (list[ComposerABVariant]):
            best_time (bool | Unset):
            confidence (float | Unset): How sure the test must be (default 0.95)
            min_sample (int | Unset): Recipients each variant needs before a winner can be picked (default 500)
            no_winner (ComposerABTestNoWinner | Unset): What the rest get without a clear winner: default, the first variant
            winner (str | Unset):
     """

    duration_hours: int
    kind: ComposerABTestKind
    metric: ComposerABTestMetric
    test_percent: int
    variants: list[ComposerABVariant]
    best_time: bool | Unset = UNSET
    confidence: float | Unset = UNSET
    min_sample: int | Unset = UNSET
    no_winner: ComposerABTestNoWinner | Unset = UNSET
    winner: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_ab_variant import ComposerABVariant # noqa: PLC0415
        duration_hours = self.duration_hours

        kind = self.kind.value

        metric = self.metric.value

        test_percent = self.test_percent

        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)



        best_time = self.best_time

        confidence = self.confidence

        min_sample = self.min_sample

        no_winner: str | Unset = UNSET
        if not isinstance(self.no_winner, Unset):
            no_winner = self.no_winner.value


        winner = self.winner


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "durationHours": duration_hours,
            "kind": kind,
            "metric": metric,
            "testPercent": test_percent,
            "variants": variants,
        })
        if best_time is not UNSET:
            field_dict["bestTime"] = best_time
        if confidence is not UNSET:
            field_dict["confidence"] = confidence
        if min_sample is not UNSET:
            field_dict["minSample"] = min_sample
        if no_winner is not UNSET:
            field_dict["noWinner"] = no_winner
        if winner is not UNSET:
            field_dict["winner"] = winner

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_ab_variant import ComposerABVariant # noqa: PLC0415
        d = dict(src_dict)
        duration_hours = d.pop("durationHours")

        kind = ComposerABTestKind(d.pop("kind"))




        metric = ComposerABTestMetric(d.pop("metric"))




        test_percent = d.pop("testPercent")

        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = ComposerABVariant.from_dict(variants_item_data)



            variants.append(variants_item)


        best_time = d.pop("bestTime", UNSET)

        confidence = d.pop("confidence", UNSET)

        min_sample = d.pop("minSample", UNSET)

        _no_winner = d.pop("noWinner", UNSET)
        no_winner: ComposerABTestNoWinner | Unset
        if isinstance(_no_winner,  Unset):
            no_winner = UNSET
        else:
            no_winner = ComposerABTestNoWinner(_no_winner)




        winner = d.pop("winner", UNSET)

        composer_ab_test = cls(
            duration_hours=duration_hours,
            kind=kind,
            metric=metric,
            test_percent=test_percent,
            variants=variants,
            best_time=best_time,
            confidence=confidence,
            min_sample=min_sample,
            no_winner=no_winner,
            winner=winner,
        )

        return composer_ab_test

