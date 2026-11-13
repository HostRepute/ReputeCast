from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SponsorMediaKit")



@_attrs_define
class SponsorMediaKit:
    """ 
        Attributes:
            about (str):
            audience (str):
            click_rate (bool):
            open_rate (bool):
            subscribers (bool):
     """

    about: str
    audience: str
    click_rate: bool
    open_rate: bool
    subscribers: bool





    def to_dict(self) -> dict[str, Any]:
        about = self.about

        audience = self.audience

        click_rate = self.click_rate

        open_rate = self.open_rate

        subscribers = self.subscribers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "about": about,
            "audience": audience,
            "clickRate": click_rate,
            "openRate": open_rate,
            "subscribers": subscribers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        about = d.pop("about")

        audience = d.pop("audience")

        click_rate = d.pop("clickRate")

        open_rate = d.pop("openRate")

        subscribers = d.pop("subscribers")

        sponsor_media_kit = cls(
            about=about,
            audience=audience,
            click_rate=click_rate,
            open_rate=open_rate,
            subscribers=subscribers,
        )

        return sponsor_media_kit

