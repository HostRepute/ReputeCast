from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WebhookRetryInput")



@_attrs_define
class WebhookRetryInput:
    """ 
        Attributes:
            webhook_id (str | Unset): Only this endpoint; empty retries every endpoint
     """

    webhook_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        webhook_id = self.webhook_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if webhook_id is not UNSET:
            field_dict["webhookId"] = webhook_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        webhook_id = d.pop("webhookId", UNSET)

        webhook_retry_input = cls(
            webhook_id=webhook_id,
        )

        return webhook_retry_input

