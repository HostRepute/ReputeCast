from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_webhook_setup_mode import SmsWebhookSetupMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsWebhookSetup")



@_attrs_define
class SmsWebhookSetup:
    """ 
        Attributes:
            mode (SmsWebhookSetupMode):
            url (str):
            error (str | Unset):
            secret (str | Unset):
     """

    mode: SmsWebhookSetupMode
    url: str
    error: str | Unset = UNSET
    secret: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        url = self.url

        error = self.error

        secret = self.secret


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "url": url,
        })
        if error is not UNSET:
            field_dict["error"] = error
        if secret is not UNSET:
            field_dict["secret"] = secret

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = SmsWebhookSetupMode(d.pop("mode"))




        url = d.pop("url")

        error = d.pop("error", UNSET)

        secret = d.pop("secret", UNSET)

        sms_webhook_setup = cls(
            mode=mode,
            url=url,
            error=error,
            secret=secret,
        )

        return sms_webhook_setup

