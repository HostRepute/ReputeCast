from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_link_row_place import ReportLinkRowPlace
from typing import cast






T = TypeVar("T", bound="ReportLinkRow")



@_attrs_define
class ReportLinkRow:
    """ 
        Attributes:
            bot_clicks (int):
            clicks (int): Human clicks, each person once per link
            labels (list[str]):
            place (ReportLinkRowPlace):
            positions (list[int]):
            share (float):
     """

    bot_clicks: int
    clicks: int
    labels: list[str]
    place: ReportLinkRowPlace
    positions: list[int]
    share: float





    def to_dict(self) -> dict[str, Any]:
        bot_clicks = self.bot_clicks

        clicks = self.clicks

        labels = self.labels



        place = self.place.value

        positions = self.positions



        share = self.share


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "botClicks": bot_clicks,
            "clicks": clicks,
            "labels": labels,
            "place": place,
            "positions": positions,
            "share": share,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bot_clicks = d.pop("botClicks")

        clicks = d.pop("clicks")

        labels = cast(list[str], d.pop("labels"))


        place = ReportLinkRowPlace(d.pop("place"))




        positions = cast(list[int], d.pop("positions"))


        share = d.pop("share")

        report_link_row = cls(
            bot_clicks=bot_clicks,
            clicks=clicks,
            labels=labels,
            place=place,
            positions=positions,
            share=share,
        )

        return report_link_row

