from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_contact_message_status import TransactionalContactMessageStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_log_template import TransactionalLogTemplate





T = TypeVar("T", bound="TransactionalContactMessage")



@_attrs_define
class TransactionalContactMessage:
    """ 
        Attributes:
            at (datetime.datetime):
            id (str):
            raw (bool):
            status (TransactionalContactMessageStatus):
            subject (str):
            category (str | Unset):
            delivery_ms (int | Unset):
            provider (str | Unset):
            template (TransactionalLogTemplate | Unset):
     """

    at: datetime.datetime
    id: str
    raw: bool
    status: TransactionalContactMessageStatus
    subject: str
    category: str | Unset = UNSET
    delivery_ms: int | Unset = UNSET
    provider: str | Unset = UNSET
    template: TransactionalLogTemplate | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_log_template import TransactionalLogTemplate # noqa: PLC0415
        at = self.at.isoformat()

        id = self.id

        raw = self.raw

        status = self.status.value

        subject = self.subject

        category = self.category

        delivery_ms = self.delivery_ms

        provider = self.provider

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "id": id,
            "raw": raw,
            "status": status,
            "subject": subject,
        })
        if category is not UNSET:
            field_dict["category"] = category
        if delivery_ms is not UNSET:
            field_dict["deliveryMs"] = delivery_ms
        if provider is not UNSET:
            field_dict["provider"] = provider
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_log_template import TransactionalLogTemplate # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        id = d.pop("id")

        raw = d.pop("raw")

        status = TransactionalContactMessageStatus(d.pop("status"))




        subject = d.pop("subject")

        category = d.pop("category", UNSET)

        delivery_ms = d.pop("deliveryMs", UNSET)

        provider = d.pop("provider", UNSET)

        _template = d.pop("template", UNSET)
        template: TransactionalLogTemplate | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = TransactionalLogTemplate.from_dict(_template)




        transactional_contact_message = cls(
            at=at,
            id=id,
            raw=raw,
            status=status,
            subject=subject,
            category=category,
            delivery_ms=delivery_ms,
            provider=provider,
            template=template,
        )

        return transactional_contact_message

