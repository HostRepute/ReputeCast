from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_disconnected_webhook import ProviderDisconnectedWebhook
from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderDisconnected")



@_attrs_define
class ProviderDisconnected:
    """ 
        Attributes:
            url (str):
            webhook (ProviderDisconnectedWebhook):
            detail (str | Unset):
     """

    url: str
    webhook: ProviderDisconnectedWebhook
    detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        url = self.url

        webhook = self.webhook.value

        detail = self.detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "url": url,
            "webhook": webhook,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        url = d.pop("url")

        webhook = ProviderDisconnectedWebhook(d.pop("webhook"))




        detail = d.pop("detail", UNSET)

        provider_disconnected = cls(
            url=url,
            webhook=webhook,
            detail=detail,
        )

        return provider_disconnected

