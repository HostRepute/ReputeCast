from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_request_detail_issue_state import SponsorRequestDetailIssueState
from ..models.sponsor_request_detail_paid_source import SponsorRequestDetailPaidSource
from ..models.sponsor_request_detail_status import SponsorRequestDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_booking_event import SponsorBookingEvent
  from ..models.sponsor_creative import SponsorCreative
  from ..models.sponsor_ref import SponsorRef
  from ..models.sponsor_request_file import SponsorRequestFile
  from ..models.sponsor_request_slot import SponsorRequestSlot





T = TypeVar("T", bound="SponsorRequestDetail")



@_attrs_define
class SponsorRequestDetail:
    """ 
        Attributes:
            activity (list[SponsorBookingEvent]):
            can_manage (bool):
            company (str):
            creative (SponsorCreative):
            currency (str):
            email (str):
            files (list[SponsorRequestFile]):
            helpers (list[str]):
            id (str):
            initials (str):
            issue (int):
            issue_on (datetime.datetime):
            issue_state (SponsorRequestDetailIssueState):
            label (str):
            list_name (str):
            name (str):
            notes (str):
            pay_days (int):
            price_cents (int):
            slot_deleted (bool):
            slot_id (str):
            slot_name (str):
            slots (list[SponsorRequestSlot]):
            status (SponsorRequestDetailStatus):
            submitted_at (datetime.datetime):
            website (str):
            approved_at (datetime.datetime | Unset):
            approved_by (str | Unset):
            closed_at (datetime.datetime | Unset):
            declined_reason (str | Unset):
            held_by (str | Unset):
            held_until (datetime.datetime | Unset):
            logo (str | Unset):
            paid_on (datetime.datetime | Unset):
            paid_source (SponsorRequestDetailPaidSource | Unset):
            payment_due_at (datetime.datetime | Unset):
            sponsor (SponsorRef | Unset):
     """

    activity: list[SponsorBookingEvent]
    can_manage: bool
    company: str
    creative: SponsorCreative
    currency: str
    email: str
    files: list[SponsorRequestFile]
    helpers: list[str]
    id: str
    initials: str
    issue: int
    issue_on: datetime.datetime
    issue_state: SponsorRequestDetailIssueState
    label: str
    list_name: str
    name: str
    notes: str
    pay_days: int
    price_cents: int
    slot_deleted: bool
    slot_id: str
    slot_name: str
    slots: list[SponsorRequestSlot]
    status: SponsorRequestDetailStatus
    submitted_at: datetime.datetime
    website: str
    approved_at: datetime.datetime | Unset = UNSET
    approved_by: str | Unset = UNSET
    closed_at: datetime.datetime | Unset = UNSET
    declined_reason: str | Unset = UNSET
    held_by: str | Unset = UNSET
    held_until: datetime.datetime | Unset = UNSET
    logo: str | Unset = UNSET
    paid_on: datetime.datetime | Unset = UNSET
    paid_source: SponsorRequestDetailPaidSource | Unset = UNSET
    payment_due_at: datetime.datetime | Unset = UNSET
    sponsor: SponsorRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_booking_event import SponsorBookingEvent # noqa: PLC0415
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        from ..models.sponsor_request_file import SponsorRequestFile # noqa: PLC0415
        from ..models.sponsor_request_slot import SponsorRequestSlot # noqa: PLC0415
        activity = []
        for activity_item_data in self.activity:
            activity_item = activity_item_data.to_dict()
            activity.append(activity_item)



        can_manage = self.can_manage

        company = self.company

        creative = self.creative.to_dict()

        currency = self.currency

        email = self.email

        files = []
        for files_item_data in self.files:
            files_item = files_item_data.to_dict()
            files.append(files_item)



        helpers = self.helpers



        id = self.id

        initials = self.initials

        issue = self.issue

        issue_on = self.issue_on.isoformat()

        issue_state = self.issue_state.value

        label = self.label

        list_name = self.list_name

        name = self.name

        notes = self.notes

        pay_days = self.pay_days

        price_cents = self.price_cents

        slot_deleted = self.slot_deleted

        slot_id = self.slot_id

        slot_name = self.slot_name

        slots = []
        for slots_item_data in self.slots:
            slots_item = slots_item_data.to_dict()
            slots.append(slots_item)



        status = self.status.value

        submitted_at = self.submitted_at.isoformat()

        website = self.website

        approved_at: str | Unset = UNSET
        if not isinstance(self.approved_at, Unset):
            approved_at = self.approved_at.isoformat()

        approved_by = self.approved_by

        closed_at: str | Unset = UNSET
        if not isinstance(self.closed_at, Unset):
            closed_at = self.closed_at.isoformat()

        declined_reason = self.declined_reason

        held_by = self.held_by

        held_until: str | Unset = UNSET
        if not isinstance(self.held_until, Unset):
            held_until = self.held_until.isoformat()

        logo = self.logo

        paid_on: str | Unset = UNSET
        if not isinstance(self.paid_on, Unset):
            paid_on = self.paid_on.isoformat()

        paid_source: str | Unset = UNSET
        if not isinstance(self.paid_source, Unset):
            paid_source = self.paid_source.value


        payment_due_at: str | Unset = UNSET
        if not isinstance(self.payment_due_at, Unset):
            payment_due_at = self.payment_due_at.isoformat()

        sponsor: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sponsor, Unset):
            sponsor = self.sponsor.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "activity": activity,
            "canManage": can_manage,
            "company": company,
            "creative": creative,
            "currency": currency,
            "email": email,
            "files": files,
            "helpers": helpers,
            "id": id,
            "initials": initials,
            "issue": issue,
            "issueOn": issue_on,
            "issueState": issue_state,
            "label": label,
            "listName": list_name,
            "name": name,
            "notes": notes,
            "payDays": pay_days,
            "priceCents": price_cents,
            "slotDeleted": slot_deleted,
            "slotId": slot_id,
            "slotName": slot_name,
            "slots": slots,
            "status": status,
            "submittedAt": submitted_at,
            "website": website,
        })
        if approved_at is not UNSET:
            field_dict["approvedAt"] = approved_at
        if approved_by is not UNSET:
            field_dict["approvedBy"] = approved_by
        if closed_at is not UNSET:
            field_dict["closedAt"] = closed_at
        if declined_reason is not UNSET:
            field_dict["declinedReason"] = declined_reason
        if held_by is not UNSET:
            field_dict["heldBy"] = held_by
        if held_until is not UNSET:
            field_dict["heldUntil"] = held_until
        if logo is not UNSET:
            field_dict["logo"] = logo
        if paid_on is not UNSET:
            field_dict["paidOn"] = paid_on
        if paid_source is not UNSET:
            field_dict["paidSource"] = paid_source
        if payment_due_at is not UNSET:
            field_dict["paymentDueAt"] = payment_due_at
        if sponsor is not UNSET:
            field_dict["sponsor"] = sponsor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_booking_event import SponsorBookingEvent # noqa: PLC0415
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        from ..models.sponsor_request_file import SponsorRequestFile # noqa: PLC0415
        from ..models.sponsor_request_slot import SponsorRequestSlot # noqa: PLC0415
        d = dict(src_dict)
        activity = []
        _activity = d.pop("activity")
        for activity_item_data in (_activity):
            activity_item = SponsorBookingEvent.from_dict(activity_item_data)



            activity.append(activity_item)


        can_manage = d.pop("canManage")

        company = d.pop("company")

        creative = SponsorCreative.from_dict(d.pop("creative"))




        currency = d.pop("currency")

        email = d.pop("email")

        files = []
        _files = d.pop("files")
        for files_item_data in (_files):
            files_item = SponsorRequestFile.from_dict(files_item_data)



            files.append(files_item)


        helpers = cast(list[str], d.pop("helpers"))


        id = d.pop("id")

        initials = d.pop("initials")

        issue = d.pop("issue")

        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        issue_state = SponsorRequestDetailIssueState(d.pop("issueState"))




        label = d.pop("label")

        list_name = d.pop("listName")

        name = d.pop("name")

        notes = d.pop("notes")

        pay_days = d.pop("payDays")

        price_cents = d.pop("priceCents")

        slot_deleted = d.pop("slotDeleted")

        slot_id = d.pop("slotId")

        slot_name = d.pop("slotName")

        slots = []
        _slots = d.pop("slots")
        for slots_item_data in (_slots):
            slots_item = SponsorRequestSlot.from_dict(slots_item_data)



            slots.append(slots_item)


        status = SponsorRequestDetailStatus(d.pop("status"))




        submitted_at = datetime.datetime.fromisoformat(d.pop("submittedAt"))




        website = d.pop("website")

        _approved_at = d.pop("approvedAt", UNSET)
        approved_at: datetime.datetime | Unset
        if isinstance(_approved_at,  Unset):
            approved_at = UNSET
        else:
            approved_at = datetime.datetime.fromisoformat(_approved_at)




        approved_by = d.pop("approvedBy", UNSET)

        _closed_at = d.pop("closedAt", UNSET)
        closed_at: datetime.datetime | Unset
        if isinstance(_closed_at,  Unset):
            closed_at = UNSET
        else:
            closed_at = datetime.datetime.fromisoformat(_closed_at)




        declined_reason = d.pop("declinedReason", UNSET)

        held_by = d.pop("heldBy", UNSET)

        _held_until = d.pop("heldUntil", UNSET)
        held_until: datetime.datetime | Unset
        if isinstance(_held_until,  Unset):
            held_until = UNSET
        else:
            held_until = datetime.datetime.fromisoformat(_held_until)




        logo = d.pop("logo", UNSET)

        _paid_on = d.pop("paidOn", UNSET)
        paid_on: datetime.datetime | Unset
        if isinstance(_paid_on,  Unset):
            paid_on = UNSET
        else:
            paid_on = datetime.datetime.fromisoformat(_paid_on)




        _paid_source = d.pop("paidSource", UNSET)
        paid_source: SponsorRequestDetailPaidSource | Unset
        if isinstance(_paid_source,  Unset):
            paid_source = UNSET
        else:
            paid_source = SponsorRequestDetailPaidSource(_paid_source)




        _payment_due_at = d.pop("paymentDueAt", UNSET)
        payment_due_at: datetime.datetime | Unset
        if isinstance(_payment_due_at,  Unset):
            payment_due_at = UNSET
        else:
            payment_due_at = datetime.datetime.fromisoformat(_payment_due_at)




        _sponsor = d.pop("sponsor", UNSET)
        sponsor: SponsorRef | Unset
        if isinstance(_sponsor,  Unset):
            sponsor = UNSET
        else:
            sponsor = SponsorRef.from_dict(_sponsor)




        sponsor_request_detail = cls(
            activity=activity,
            can_manage=can_manage,
            company=company,
            creative=creative,
            currency=currency,
            email=email,
            files=files,
            helpers=helpers,
            id=id,
            initials=initials,
            issue=issue,
            issue_on=issue_on,
            issue_state=issue_state,
            label=label,
            list_name=list_name,
            name=name,
            notes=notes,
            pay_days=pay_days,
            price_cents=price_cents,
            slot_deleted=slot_deleted,
            slot_id=slot_id,
            slot_name=slot_name,
            slots=slots,
            status=status,
            submitted_at=submitted_at,
            website=website,
            approved_at=approved_at,
            approved_by=approved_by,
            closed_at=closed_at,
            declined_reason=declined_reason,
            held_by=held_by,
            held_until=held_until,
            logo=logo,
            paid_on=paid_on,
            paid_source=paid_source,
            payment_due_at=payment_due_at,
            sponsor=sponsor,
        )

        return sponsor_request_detail

