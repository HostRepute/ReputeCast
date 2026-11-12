from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_payments_mode import WebsitePaymentsMode






T = TypeVar("T", bound="WebsitePayments")



@_attrs_define
class WebsitePayments:
    """ 
        Attributes:
            account (str):
            connected (bool):
            mode (WebsitePaymentsMode):
     """

    account: str
    connected: bool
    mode: WebsitePaymentsMode





    def to_dict(self) -> dict[str, Any]:
        account = self.account

        connected = self.connected

        mode = self.mode.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "connected": connected,
            "mode": mode,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        account = d.pop("account")

        connected = d.pop("connected")

        mode = WebsitePaymentsMode(d.pop("mode"))




        website_payments = cls(
            account=account,
            connected=connected,
            mode=mode,
        )

        return website_payments

