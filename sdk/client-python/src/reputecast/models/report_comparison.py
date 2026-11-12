from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_comparison_metric import ReportComparisonMetric
from ..models.report_comparison_tone import ReportComparisonTone
from ..models.report_comparison_unit import ReportComparisonUnit
from typing import cast






T = TypeVar("T", bound="ReportComparison")



@_attrs_define
class ReportComparison:
    """ 
        Attributes:
            change (float):
            median (float):
            metric (ReportComparisonMetric):
            tone (ReportComparisonTone):
            trend (list[float]):
            unit (ReportComparisonUnit):
     """

    change: float
    median: float
    metric: ReportComparisonMetric
    tone: ReportComparisonTone
    trend: list[float]
    unit: ReportComparisonUnit





    def to_dict(self) -> dict[str, Any]:
        change = self.change

        median = self.median

        metric = self.metric.value

        tone = self.tone.value

        trend = self.trend



        unit = self.unit.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "change": change,
            "median": median,
            "metric": metric,
            "tone": tone,
            "trend": trend,
            "unit": unit,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        change = d.pop("change")

        median = d.pop("median")

        metric = ReportComparisonMetric(d.pop("metric"))




        tone = ReportComparisonTone(d.pop("tone"))




        trend = cast(list[float], d.pop("trend"))


        unit = ReportComparisonUnit(d.pop("unit"))




        report_comparison = cls(
            change=change,
            median=median,
            metric=metric,
            tone=tone,
            trend=trend,
            unit=unit,
        )

        return report_comparison

