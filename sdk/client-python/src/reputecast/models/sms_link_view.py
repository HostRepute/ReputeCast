from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsLinkView")



@_attrs_define
class SmsLinkView:
    """ 
        Attributes:
            host (str):
            length (int):
            shortener (bool):
            url (str):
            utm_campaign (str):
     """

    host: str
    length: int
    shortener: bool
    url: str
    utm_campaign: str





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        length = self.length

        shortener = self.shortener

        url = self.url

        utm_campaign = self.utm_campaign


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "length": length,
            "shortener": shortener,
            "url": url,
            "utmCampaign": utm_campaign,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        length = d.pop("length")

        shortener = d.pop("shortener")

        url = d.pop("url")

        utm_campaign = d.pop("utmCampaign")

        sms_link_view = cls(
            host=host,
            length=length,
            shortener=shortener,
            url=url,
            utm_campaign=utm_campaign,
        )

        return sms_link_view

