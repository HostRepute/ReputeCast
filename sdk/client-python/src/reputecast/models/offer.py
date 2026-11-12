from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.offer_discount_type import OfferDiscountType
from ..models.offer_duration import OfferDuration
from ..models.offer_ended_reason import OfferEndedReason
from ..models.offer_kind import OfferKind
from ..models.offer_state import OfferState
from ..models.offer_status import OfferStatus
from ..models.offer_sync import OfferSync
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.offer_applies import OfferApplies
  from ..models.offer_results import OfferResults
  from ..models.offer_stripe import OfferStripe





T = TypeVar("T", bound="Offer")



@_attrs_define
class Offer:
    """ 
        Attributes:
            applies (list[OfferApplies]):
            by_code (bool):
            by_link (bool):
            code (str):
            code_days (int):
            currency (str):
            discount_type (OfferDiscountType):
            duration (OfferDuration):
            id (str):
            kind (OfferKind):
            min_paid_months (int):
            name (str):
            new_only (bool):
            results (OfferResults):
            started_at (datetime.datetime):
            state (OfferState):
            status (OfferStatus):
            stripe (OfferStripe):
            sync (OfferSync):
            amount_off_cents (int | Unset):
            ended_at (datetime.datetime | Unset):
            ended_by (str | Unset):
            ended_reason (OfferEndedReason | Unset):
            ends_at (datetime.datetime | Unset):
            limit (int | Unset):
            link (str | Unset):
            link_host (str | Unset):
            months (int | Unset):
            percent_off (int | Unset):
            started_by (str | Unset):
            sync_error (str | Unset):
     """

    applies: list[OfferApplies]
    by_code: bool
    by_link: bool
    code: str
    code_days: int
    currency: str
    discount_type: OfferDiscountType
    duration: OfferDuration
    id: str
    kind: OfferKind
    min_paid_months: int
    name: str
    new_only: bool
    results: OfferResults
    started_at: datetime.datetime
    state: OfferState
    status: OfferStatus
    stripe: OfferStripe
    sync: OfferSync
    amount_off_cents: int | Unset = UNSET
    ended_at: datetime.datetime | Unset = UNSET
    ended_by: str | Unset = UNSET
    ended_reason: OfferEndedReason | Unset = UNSET
    ends_at: datetime.datetime | Unset = UNSET
    limit: int | Unset = UNSET
    link: str | Unset = UNSET
    link_host: str | Unset = UNSET
    months: int | Unset = UNSET
    percent_off: int | Unset = UNSET
    started_by: str | Unset = UNSET
    sync_error: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.offer_applies import OfferApplies # noqa: PLC0415
        from ..models.offer_results import OfferResults # noqa: PLC0415
        from ..models.offer_stripe import OfferStripe # noqa: PLC0415
        applies = []
        for applies_item_data in self.applies:
            applies_item = applies_item_data.to_dict()
            applies.append(applies_item)



        by_code = self.by_code

        by_link = self.by_link

        code = self.code

        code_days = self.code_days

        currency = self.currency

        discount_type = self.discount_type.value

        duration = self.duration.value

        id = self.id

        kind = self.kind.value

        min_paid_months = self.min_paid_months

        name = self.name

        new_only = self.new_only

        results = self.results.to_dict()

        started_at = self.started_at.isoformat()

        state = self.state.value

        status = self.status.value

        stripe = self.stripe.to_dict()

        sync = self.sync.value

        amount_off_cents = self.amount_off_cents

        ended_at: str | Unset = UNSET
        if not isinstance(self.ended_at, Unset):
            ended_at = self.ended_at.isoformat()

        ended_by = self.ended_by

        ended_reason: str | Unset = UNSET
        if not isinstance(self.ended_reason, Unset):
            ended_reason = self.ended_reason.value


        ends_at: str | Unset = UNSET
        if not isinstance(self.ends_at, Unset):
            ends_at = self.ends_at.isoformat()

        limit = self.limit

        link = self.link

        link_host = self.link_host

        months = self.months

        percent_off = self.percent_off

        started_by = self.started_by

        sync_error = self.sync_error


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "applies": applies,
            "byCode": by_code,
            "byLink": by_link,
            "code": code,
            "codeDays": code_days,
            "currency": currency,
            "discountType": discount_type,
            "duration": duration,
            "id": id,
            "kind": kind,
            "minPaidMonths": min_paid_months,
            "name": name,
            "newOnly": new_only,
            "results": results,
            "startedAt": started_at,
            "state": state,
            "status": status,
            "stripe": stripe,
            "sync": sync,
        })
        if amount_off_cents is not UNSET:
            field_dict["amountOffCents"] = amount_off_cents
        if ended_at is not UNSET:
            field_dict["endedAt"] = ended_at
        if ended_by is not UNSET:
            field_dict["endedBy"] = ended_by
        if ended_reason is not UNSET:
            field_dict["endedReason"] = ended_reason
        if ends_at is not UNSET:
            field_dict["endsAt"] = ends_at
        if limit is not UNSET:
            field_dict["limit"] = limit
        if link is not UNSET:
            field_dict["link"] = link
        if link_host is not UNSET:
            field_dict["linkHost"] = link_host
        if months is not UNSET:
            field_dict["months"] = months
        if percent_off is not UNSET:
            field_dict["percentOff"] = percent_off
        if started_by is not UNSET:
            field_dict["startedBy"] = started_by
        if sync_error is not UNSET:
            field_dict["syncError"] = sync_error

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.offer_applies import OfferApplies # noqa: PLC0415
        from ..models.offer_results import OfferResults # noqa: PLC0415
        from ..models.offer_stripe import OfferStripe # noqa: PLC0415
        d = dict(src_dict)
        applies = []
        _applies = d.pop("applies")
        for applies_item_data in (_applies):
            applies_item = OfferApplies.from_dict(applies_item_data)



            applies.append(applies_item)


        by_code = d.pop("byCode")

        by_link = d.pop("byLink")

        code = d.pop("code")

        code_days = d.pop("codeDays")

        currency = d.pop("currency")

        discount_type = OfferDiscountType(d.pop("discountType"))




        duration = OfferDuration(d.pop("duration"))




        id = d.pop("id")

        kind = OfferKind(d.pop("kind"))




        min_paid_months = d.pop("minPaidMonths")

        name = d.pop("name")

        new_only = d.pop("newOnly")

        results = OfferResults.from_dict(d.pop("results"))




        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        state = OfferState(d.pop("state"))




        status = OfferStatus(d.pop("status"))




        stripe = OfferStripe.from_dict(d.pop("stripe"))




        sync = OfferSync(d.pop("sync"))




        amount_off_cents = d.pop("amountOffCents", UNSET)

        _ended_at = d.pop("endedAt", UNSET)
        ended_at: datetime.datetime | Unset
        if isinstance(_ended_at,  Unset):
            ended_at = UNSET
        else:
            ended_at = datetime.datetime.fromisoformat(_ended_at)




        ended_by = d.pop("endedBy", UNSET)

        _ended_reason = d.pop("endedReason", UNSET)
        ended_reason: OfferEndedReason | Unset
        if isinstance(_ended_reason,  Unset):
            ended_reason = UNSET
        else:
            ended_reason = OfferEndedReason(_ended_reason)




        _ends_at = d.pop("endsAt", UNSET)
        ends_at: datetime.datetime | Unset
        if isinstance(_ends_at,  Unset):
            ends_at = UNSET
        else:
            ends_at = datetime.datetime.fromisoformat(_ends_at)




        limit = d.pop("limit", UNSET)

        link = d.pop("link", UNSET)

        link_host = d.pop("linkHost", UNSET)

        months = d.pop("months", UNSET)

        percent_off = d.pop("percentOff", UNSET)

        started_by = d.pop("startedBy", UNSET)

        sync_error = d.pop("syncError", UNSET)

        offer = cls(
            applies=applies,
            by_code=by_code,
            by_link=by_link,
            code=code,
            code_days=code_days,
            currency=currency,
            discount_type=discount_type,
            duration=duration,
            id=id,
            kind=kind,
            min_paid_months=min_paid_months,
            name=name,
            new_only=new_only,
            results=results,
            started_at=started_at,
            state=state,
            status=status,
            stripe=stripe,
            sync=sync,
            amount_off_cents=amount_off_cents,
            ended_at=ended_at,
            ended_by=ended_by,
            ended_reason=ended_reason,
            ends_at=ends_at,
            limit=limit,
            link=link,
            link_host=link_host,
            months=months,
            percent_off=percent_off,
            started_by=started_by,
            sync_error=sync_error,
        )

        return offer

