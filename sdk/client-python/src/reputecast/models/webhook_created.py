from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.webhook_view import WebhookView





T = TypeVar("T", bound="WebhookCreated")



@_attrs_define
class WebhookCreated:
    """ 
        Attributes:
            secret (str): The signing secret. Shown only in this response.
            webhook (WebhookView):
     """

    secret: str
    webhook: WebhookView





    def to_dict(self) -> dict[str, Any]:
        from ..models.webhook_view import WebhookView # noqa: PLC0415
        secret = self.secret

        webhook = self.webhook.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "secret": secret,
            "webhook": webhook,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_view import WebhookView # noqa: PLC0415
        d = dict(src_dict)
        secret = d.pop("secret")

        webhook = WebhookView.from_dict(d.pop("webhook"))




        webhook_created = cls(
            secret=secret,
            webhook=webhook,
        )

        return webhook_created

