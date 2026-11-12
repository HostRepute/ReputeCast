from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.growth_chart_day import GrowthChartDay
  from ..models.growth_series import GrowthSeries





T = TypeVar("T", bound="GrowthChart")



@_attrs_define
class GrowthChart:
    """ 
        Attributes:
            days (list[GrowthChartDay]):
            series (list[GrowthSeries]):
     """

    days: list[GrowthChartDay]
    series: list[GrowthSeries]





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_chart_day import GrowthChartDay # noqa: PLC0415
        from ..models.growth_series import GrowthSeries # noqa: PLC0415
        days = []
        for days_item_data in self.days:
            days_item = days_item_data.to_dict()
            days.append(days_item)



        series = []
        for series_item_data in self.series:
            series_item = series_item_data.to_dict()
            series.append(series_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "series": series,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_chart_day import GrowthChartDay # noqa: PLC0415
        from ..models.growth_series import GrowthSeries # noqa: PLC0415
        d = dict(src_dict)
        days = []
        _days = d.pop("days")
        for days_item_data in (_days):
            days_item = GrowthChartDay.from_dict(days_item_data)



            days.append(days_item)


        series = []
        _series = d.pop("series")
        for series_item_data in (_series):
            series_item = GrowthSeries.from_dict(series_item_data)



            series.append(series_item)


        growth_chart = cls(
            days=days,
            series=series,
        )

        return growth_chart

