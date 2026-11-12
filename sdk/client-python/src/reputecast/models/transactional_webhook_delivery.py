from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalWebhookDelivery")



@_attrs_define
class TransactionalWebhookDelivery:
    """ 
        Attributes:
            at (datetime.datetime):
            event (str):
            state (str):
            url (str):
            delivered_at (datetime.datetime | Unset):
            status_code (int | Unset):
     """

    at: datetime.datetime
    event: str
    state: str
    url: str
    delivered_at: datetime.datetime | Unset = UNSET
    status_code: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        event = self.event

        state = self.state

        url = self.url

        delivered_at: str | Unset = UNSET
        if not isinstance(self.delivered_at, Unset):
            delivered_at = self.delivered_at.isoformat()

        status_code = self.status_code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "event": event,
            "state": state,
            "url": url,
        })
        if delivered_at is not UNSET:
            field_dict["deliveredAt"] = delivered_at
        if status_code is not UNSET:
            field_dict["statusCode"] = status_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        event = d.pop("event")

        state = d.pop("state")

        url = d.pop("url")

        _delivered_at = d.pop("deliveredAt", UNSET)
        delivered_at: datetime.datetime | Unset
        if isinstance(_delivered_at,  Unset):
            delivered_at = UNSET
        else:
            delivered_at = datetime.datetime.fromisoformat(_delivered_at)




        status_code = d.pop("statusCode", UNSET)

        transactional_webhook_delivery = cls(
            at=at,
            event=event,
            state=state,
            url=url,
            delivered_at=delivered_at,
            status_code=status_code,
        )

        return transactional_webhook_delivery

