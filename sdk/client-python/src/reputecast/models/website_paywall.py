from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WebsitePaywall")



@_attrs_define
class WebsitePaywall:
    """ 
        Attributes:
            free (str):
            heading (str):
            signup (str):
            text (str):
            tiers (list[str]):
            upgrade (str):
     """

    free: str
    heading: str
    signup: str
    text: str
    tiers: list[str]
    upgrade: str





    def to_dict(self) -> dict[str, Any]:
        free = self.free

        heading = self.heading

        signup = self.signup

        text = self.text

        tiers = self.tiers



        upgrade = self.upgrade


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "free": free,
            "heading": heading,
            "signup": signup,
            "text": text,
            "tiers": tiers,
            "upgrade": upgrade,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        free = d.pop("free")

        heading = d.pop("heading")

        signup = d.pop("signup")

        text = d.pop("text")

        tiers = cast(list[str], d.pop("tiers"))


        upgrade = d.pop("upgrade")

        website_paywall = cls(
            free=free,
            heading=heading,
            signup=signup,
            text=text,
            tiers=tiers,
            upgrade=upgrade,
        )

        return website_paywall

