from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_event_mode import EarnEventMode
from ..models.earn_event_source import EarnEventSource
from ..models.earn_event_status import EarnEventStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_event_detail import EarnEventDetail





T = TypeVar("T", bound="EarnEvent")



@_attrs_define
class EarnEvent:
    """ 
        Attributes:
            detail (EarnEventDetail):
            event_id (str):
            id (str):
            mode (EarnEventMode):
            occurred_at (datetime.datetime):
            outcome (str):
            received_at (datetime.datetime):
            source (EarnEventSource):
            status (EarnEventStatus):
            type_ (str):
            amount_cents (int | Unset):
            contact_id (str | Unset):
            currency (str | Unset):
            event_url (str | Unset):
            member_email (str | Unset):
            member_id (str | Unset):
            member_name (str | Unset):
     """

    detail: EarnEventDetail
    event_id: str
    id: str
    mode: EarnEventMode
    occurred_at: datetime.datetime
    outcome: str
    received_at: datetime.datetime
    source: EarnEventSource
    status: EarnEventStatus
    type_: str
    amount_cents: int | Unset = UNSET
    contact_id: str | Unset = UNSET
    currency: str | Unset = UNSET
    event_url: str | Unset = UNSET
    member_email: str | Unset = UNSET
    member_id: str | Unset = UNSET
    member_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_event_detail import EarnEventDetail # noqa: PLC0415
        detail = self.detail.to_dict()

        event_id = self.event_id

        id = self.id

        mode = self.mode.value

        occurred_at = self.occurred_at.isoformat()

        outcome = self.outcome

        received_at = self.received_at.isoformat()

        source = self.source.value

        status = self.status.value

        type_ = self.type_

        amount_cents = self.amount_cents

        contact_id = self.contact_id

        currency = self.currency

        event_url = self.event_url

        member_email = self.member_email

        member_id = self.member_id

        member_name = self.member_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "eventId": event_id,
            "id": id,
            "mode": mode,
            "occurredAt": occurred_at,
            "outcome": outcome,
            "receivedAt": received_at,
            "source": source,
            "status": status,
            "type": type_,
        })
        if amount_cents is not UNSET:
            field_dict["amountCents"] = amount_cents
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if currency is not UNSET:
            field_dict["currency"] = currency
        if event_url is not UNSET:
            field_dict["eventUrl"] = event_url
        if member_email is not UNSET:
            field_dict["memberEmail"] = member_email
        if member_id is not UNSET:
            field_dict["memberId"] = member_id
        if member_name is not UNSET:
            field_dict["memberName"] = member_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_event_detail import EarnEventDetail # noqa: PLC0415
        d = dict(src_dict)
        detail = EarnEventDetail.from_dict(d.pop("detail"))




        event_id = d.pop("eventId")

        id = d.pop("id")

        mode = EarnEventMode(d.pop("mode"))




        occurred_at = datetime.datetime.fromisoformat(d.pop("occurredAt"))




        outcome = d.pop("outcome")

        received_at = datetime.datetime.fromisoformat(d.pop("receivedAt"))




        source = EarnEventSource(d.pop("source"))




        status = EarnEventStatus(d.pop("status"))




        type_ = d.pop("type")

        amount_cents = d.pop("amountCents", UNSET)

        contact_id = d.pop("contactId", UNSET)

        currency = d.pop("currency", UNSET)

        event_url = d.pop("eventUrl", UNSET)

        member_email = d.pop("memberEmail", UNSET)

        member_id = d.pop("memberId", UNSET)

        member_name = d.pop("memberName", UNSET)

        earn_event = cls(
            detail=detail,
            event_id=event_id,
            id=id,
            mode=mode,
            occurred_at=occurred_at,
            outcome=outcome,
            received_at=received_at,
            source=source,
            status=status,
            type_=type_,
            amount_cents=amount_cents,
            contact_id=contact_id,
            currency=currency,
            event_url=event_url,
            member_email=member_email,
            member_id=member_id,
            member_name=member_name,
        )

        return earn_event

