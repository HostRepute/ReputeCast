from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_log_item_status import TransactionalLogItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_bounce import TransactionalBounce
  from ..models.transactional_failover import TransactionalFailover
  from ..models.transactional_log_template import TransactionalLogTemplate
  from ..models.transactional_ref import TransactionalRef
  from ..models.transactional_rejection import TransactionalRejection





T = TypeVar("T", bound="TransactionalLogItem")



@_attrs_define
class TransactionalLogItem:
    """ 
        Attributes:
            at (datetime.datetime): When the call was accepted
            deferrals (int):
            email (str):
            id (str):
            raw (bool):
            role_address (bool):
            status (TransactionalLogItemStatus):
            subject (str):
            test_delivered (bool):
            tracked (bool):
            bounce (TransactionalBounce | Unset):
            category (str | Unset):
            clicked_at (datetime.datetime | Unset):
            contact_id (str | Unset):
            delivery_ms (int | Unset):
            error (str | Unset):
            failed_over (TransactionalFailover | Unset):
            key (str | Unset):
            language (str | Unset):
            name (str | Unset):
            opened_at (datetime.datetime | Unset):
            provider (TransactionalRef | Unset):
            rejection (TransactionalRejection | Unset):
            template (TransactionalLogTemplate | Unset):
     """

    at: datetime.datetime
    deferrals: int
    email: str
    id: str
    raw: bool
    role_address: bool
    status: TransactionalLogItemStatus
    subject: str
    test_delivered: bool
    tracked: bool
    bounce: TransactionalBounce | Unset = UNSET
    category: str | Unset = UNSET
    clicked_at: datetime.datetime | Unset = UNSET
    contact_id: str | Unset = UNSET
    delivery_ms: int | Unset = UNSET
    error: str | Unset = UNSET
    failed_over: TransactionalFailover | Unset = UNSET
    key: str | Unset = UNSET
    language: str | Unset = UNSET
    name: str | Unset = UNSET
    opened_at: datetime.datetime | Unset = UNSET
    provider: TransactionalRef | Unset = UNSET
    rejection: TransactionalRejection | Unset = UNSET
    template: TransactionalLogTemplate | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_bounce import TransactionalBounce # noqa: PLC0415
        from ..models.transactional_failover import TransactionalFailover # noqa: PLC0415
        from ..models.transactional_log_template import TransactionalLogTemplate # noqa: PLC0415
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        from ..models.transactional_rejection import TransactionalRejection # noqa: PLC0415
        at = self.at.isoformat()

        deferrals = self.deferrals

        email = self.email

        id = self.id

        raw = self.raw

        role_address = self.role_address

        status = self.status.value

        subject = self.subject

        test_delivered = self.test_delivered

        tracked = self.tracked

        bounce: dict[str, Any] | Unset = UNSET
        if not isinstance(self.bounce, Unset):
            bounce = self.bounce.to_dict()

        category = self.category

        clicked_at: str | Unset = UNSET
        if not isinstance(self.clicked_at, Unset):
            clicked_at = self.clicked_at.isoformat()

        contact_id = self.contact_id

        delivery_ms = self.delivery_ms

        error = self.error

        failed_over: dict[str, Any] | Unset = UNSET
        if not isinstance(self.failed_over, Unset):
            failed_over = self.failed_over.to_dict()

        key = self.key

        language = self.language

        name = self.name

        opened_at: str | Unset = UNSET
        if not isinstance(self.opened_at, Unset):
            opened_at = self.opened_at.isoformat()

        provider: dict[str, Any] | Unset = UNSET
        if not isinstance(self.provider, Unset):
            provider = self.provider.to_dict()

        rejection: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rejection, Unset):
            rejection = self.rejection.to_dict()

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "deferrals": deferrals,
            "email": email,
            "id": id,
            "raw": raw,
            "roleAddress": role_address,
            "status": status,
            "subject": subject,
            "testDelivered": test_delivered,
            "tracked": tracked,
        })
        if bounce is not UNSET:
            field_dict["bounce"] = bounce
        if category is not UNSET:
            field_dict["category"] = category
        if clicked_at is not UNSET:
            field_dict["clickedAt"] = clicked_at
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if delivery_ms is not UNSET:
            field_dict["deliveryMs"] = delivery_ms
        if error is not UNSET:
            field_dict["error"] = error
        if failed_over is not UNSET:
            field_dict["failedOver"] = failed_over
        if key is not UNSET:
            field_dict["key"] = key
        if language is not UNSET:
            field_dict["language"] = language
        if name is not UNSET:
            field_dict["name"] = name
        if opened_at is not UNSET:
            field_dict["openedAt"] = opened_at
        if provider is not UNSET:
            field_dict["provider"] = provider
        if rejection is not UNSET:
            field_dict["rejection"] = rejection
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_bounce import TransactionalBounce # noqa: PLC0415
        from ..models.transactional_failover import TransactionalFailover # noqa: PLC0415
        from ..models.transactional_log_template import TransactionalLogTemplate # noqa: PLC0415
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        from ..models.transactional_rejection import TransactionalRejection # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        deferrals = d.pop("deferrals")

        email = d.pop("email")

        id = d.pop("id")

        raw = d.pop("raw")

        role_address = d.pop("roleAddress")

        status = TransactionalLogItemStatus(d.pop("status"))




        subject = d.pop("subject")

        test_delivered = d.pop("testDelivered")

        tracked = d.pop("tracked")

        _bounce = d.pop("bounce", UNSET)
        bounce: TransactionalBounce | Unset
        if isinstance(_bounce,  Unset):
            bounce = UNSET
        else:
            bounce = TransactionalBounce.from_dict(_bounce)




        category = d.pop("category", UNSET)

        _clicked_at = d.pop("clickedAt", UNSET)
        clicked_at: datetime.datetime | Unset
        if isinstance(_clicked_at,  Unset):
            clicked_at = UNSET
        else:
            clicked_at = datetime.datetime.fromisoformat(_clicked_at)




        contact_id = d.pop("contactId", UNSET)

        delivery_ms = d.pop("deliveryMs", UNSET)

        error = d.pop("error", UNSET)

        _failed_over = d.pop("failedOver", UNSET)
        failed_over: TransactionalFailover | Unset
        if isinstance(_failed_over,  Unset):
            failed_over = UNSET
        else:
            failed_over = TransactionalFailover.from_dict(_failed_over)




        key = d.pop("key", UNSET)

        language = d.pop("language", UNSET)

        name = d.pop("name", UNSET)

        _opened_at = d.pop("openedAt", UNSET)
        opened_at: datetime.datetime | Unset
        if isinstance(_opened_at,  Unset):
            opened_at = UNSET
        else:
            opened_at = datetime.datetime.fromisoformat(_opened_at)




        _provider = d.pop("provider", UNSET)
        provider: TransactionalRef | Unset
        if isinstance(_provider,  Unset):
            provider = UNSET
        else:
            provider = TransactionalRef.from_dict(_provider)




        _rejection = d.pop("rejection", UNSET)
        rejection: TransactionalRejection | Unset
        if isinstance(_rejection,  Unset):
            rejection = UNSET
        else:
            rejection = TransactionalRejection.from_dict(_rejection)




        _template = d.pop("template", UNSET)
        template: TransactionalLogTemplate | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = TransactionalLogTemplate.from_dict(_template)




        transactional_log_item = cls(
            at=at,
            deferrals=deferrals,
            email=email,
            id=id,
            raw=raw,
            role_address=role_address,
            status=status,
            subject=subject,
            test_delivered=test_delivered,
            tracked=tracked,
            bounce=bounce,
            category=category,
            clicked_at=clicked_at,
            contact_id=contact_id,
            delivery_ms=delivery_ms,
            error=error,
            failed_over=failed_over,
            key=key,
            language=language,
            name=name,
            opened_at=opened_at,
            provider=provider,
            rejection=rejection,
            template=template,
        )

        return transactional_log_item

