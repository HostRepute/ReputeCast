from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportLink")



@_attrs_define
class ReportLink:
    """ 
        Attributes:
            bot_clicks (int):
            clicks (int):
            grouped (bool):
            hot (bool):
            id (str):
            label (str):
            position (int):
            share (float):
            unique_clicks (int):
            url (str):
     """

    bot_clicks: int
    clicks: int
    grouped: bool
    hot: bool
    id: str
    label: str
    position: int
    share: float
    unique_clicks: int
    url: str





    def to_dict(self) -> dict[str, Any]:
        bot_clicks = self.bot_clicks

        clicks = self.clicks

        grouped = self.grouped

        hot = self.hot

        id = self.id

        label = self.label

        position = self.position

        share = self.share

        unique_clicks = self.unique_clicks

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "botClicks": bot_clicks,
            "clicks": clicks,
            "grouped": grouped,
            "hot": hot,
            "id": id,
            "label": label,
            "position": position,
            "share": share,
            "uniqueClicks": unique_clicks,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bot_clicks = d.pop("botClicks")

        clicks = d.pop("clicks")

        grouped = d.pop("grouped")

        hot = d.pop("hot")

        id = d.pop("id")

        label = d.pop("label")

        position = d.pop("position")

        share = d.pop("share")

        unique_clicks = d.pop("uniqueClicks")

        url = d.pop("url")

        report_link = cls(
            bot_clicks=bot_clicks,
            clicks=clicks,
            grouped=grouped,
            hot=hot,
            id=id,
            label=label,
            position=position,
            share=share,
            unique_clicks=unique_clicks,
            url=url,
        )

        return report_link

