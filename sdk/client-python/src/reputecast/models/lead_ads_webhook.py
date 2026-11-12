from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LeadAdsWebhook")



@_attrs_define
class LeadAdsWebhook:
    """ 
        Attributes:
            callback_url (str):
            verify_token (str):
     """

    callback_url: str
    verify_token: str





    def to_dict(self) -> dict[str, Any]:
        callback_url = self.callback_url

        verify_token = self.verify_token


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "callbackUrl": callback_url,
            "verifyToken": verify_token,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        callback_url = d.pop("callbackUrl")

        verify_token = d.pop("verifyToken")

        lead_ads_webhook = cls(
            callback_url=callback_url,
            verify_token=verify_token,
        )

        return lead_ads_webhook

