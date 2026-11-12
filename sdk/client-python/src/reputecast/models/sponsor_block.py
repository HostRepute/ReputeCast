from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_block_booking_status import SponsorBlockBookingStatus
from ..models.sponsor_block_label_mode import SponsorBlockLabelMode
from ..models.sponsor_block_layout import SponsorBlockLayout
from ..models.sponsor_block_state import SponsorBlockState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_creative import SponsorCreative





T = TypeVar("T", bound="SponsorBlock")



@_attrs_define
class SponsorBlock:
    """ 
        Attributes:
            can_see_price (bool):
            creative (SponsorCreative):
            hash_ (str):
            helpers (list[str]):
            label (str):
            label_mode (SponsorBlockLabelMode):
            layout (SponsorBlockLayout):
            link (str):
            mjml (str):
            state (SponsorBlockState):
            booking_id (str | Unset):
            booking_status (SponsorBlockBookingStatus | Unset):
            creative_due (datetime.datetime | Unset):
            currency (str | Unset):
            initials (str | Unset):
            issue (int | Unset):
            issue_on (datetime.datetime | Unset):
            list_name (str | Unset):
            price_cents (int | Unset):
            slot_id (str | Unset):
            slot_name (str | Unset):
            sponsor (str | Unset):
     """

    can_see_price: bool
    creative: SponsorCreative
    hash_: str
    helpers: list[str]
    label: str
    label_mode: SponsorBlockLabelMode
    layout: SponsorBlockLayout
    link: str
    mjml: str
    state: SponsorBlockState
    booking_id: str | Unset = UNSET
    booking_status: SponsorBlockBookingStatus | Unset = UNSET
    creative_due: datetime.datetime | Unset = UNSET
    currency: str | Unset = UNSET
    initials: str | Unset = UNSET
    issue: int | Unset = UNSET
    issue_on: datetime.datetime | Unset = UNSET
    list_name: str | Unset = UNSET
    price_cents: int | Unset = UNSET
    slot_id: str | Unset = UNSET
    slot_name: str | Unset = UNSET
    sponsor: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        can_see_price = self.can_see_price

        creative = self.creative.to_dict()

        hash_ = self.hash_

        helpers = self.helpers



        label = self.label

        label_mode = self.label_mode.value

        layout = self.layout.value

        link = self.link

        mjml = self.mjml

        state = self.state.value

        booking_id = self.booking_id

        booking_status: str | Unset = UNSET
        if not isinstance(self.booking_status, Unset):
            booking_status = self.booking_status.value


        creative_due: str | Unset = UNSET
        if not isinstance(self.creative_due, Unset):
            creative_due = self.creative_due.isoformat()

        currency = self.currency

        initials = self.initials

        issue = self.issue

        issue_on: str | Unset = UNSET
        if not isinstance(self.issue_on, Unset):
            issue_on = self.issue_on.isoformat()

        list_name = self.list_name

        price_cents = self.price_cents

        slot_id = self.slot_id

        slot_name = self.slot_name

        sponsor = self.sponsor


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canSeePrice": can_see_price,
            "creative": creative,
            "hash": hash_,
            "helpers": helpers,
            "label": label,
            "labelMode": label_mode,
            "layout": layout,
            "link": link,
            "mjml": mjml,
            "state": state,
        })
        if booking_id is not UNSET:
            field_dict["bookingId"] = booking_id
        if booking_status is not UNSET:
            field_dict["bookingStatus"] = booking_status
        if creative_due is not UNSET:
            field_dict["creativeDue"] = creative_due
        if currency is not UNSET:
            field_dict["currency"] = currency
        if initials is not UNSET:
            field_dict["initials"] = initials
        if issue is not UNSET:
            field_dict["issue"] = issue
        if issue_on is not UNSET:
            field_dict["issueOn"] = issue_on
        if list_name is not UNSET:
            field_dict["listName"] = list_name
        if price_cents is not UNSET:
            field_dict["priceCents"] = price_cents
        if slot_id is not UNSET:
            field_dict["slotId"] = slot_id
        if slot_name is not UNSET:
            field_dict["slotName"] = slot_name
        if sponsor is not UNSET:
            field_dict["sponsor"] = sponsor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_creative import SponsorCreative # noqa: PLC0415
        d = dict(src_dict)
        can_see_price = d.pop("canSeePrice")

        creative = SponsorCreative.from_dict(d.pop("creative"))




        hash_ = d.pop("hash")

        helpers = cast(list[str], d.pop("helpers"))


        label = d.pop("label")

        label_mode = SponsorBlockLabelMode(d.pop("labelMode"))




        layout = SponsorBlockLayout(d.pop("layout"))




        link = d.pop("link")

        mjml = d.pop("mjml")

        state = SponsorBlockState(d.pop("state"))




        booking_id = d.pop("bookingId", UNSET)

        _booking_status = d.pop("bookingStatus", UNSET)
        booking_status: SponsorBlockBookingStatus | Unset
        if isinstance(_booking_status,  Unset):
            booking_status = UNSET
        else:
            booking_status = SponsorBlockBookingStatus(_booking_status)




        _creative_due = d.pop("creativeDue", UNSET)
        creative_due: datetime.datetime | Unset
        if isinstance(_creative_due,  Unset):
            creative_due = UNSET
        else:
            creative_due = datetime.datetime.fromisoformat(_creative_due)




        currency = d.pop("currency", UNSET)

        initials = d.pop("initials", UNSET)

        issue = d.pop("issue", UNSET)

        _issue_on = d.pop("issueOn", UNSET)
        issue_on: datetime.datetime | Unset
        if isinstance(_issue_on,  Unset):
            issue_on = UNSET
        else:
            issue_on = datetime.datetime.fromisoformat(_issue_on)




        list_name = d.pop("listName", UNSET)

        price_cents = d.pop("priceCents", UNSET)

        slot_id = d.pop("slotId", UNSET)

        slot_name = d.pop("slotName", UNSET)

        sponsor = d.pop("sponsor", UNSET)

        sponsor_block = cls(
            can_see_price=can_see_price,
            creative=creative,
            hash_=hash_,
            helpers=helpers,
            label=label,
            label_mode=label_mode,
            layout=layout,
            link=link,
            mjml=mjml,
            state=state,
            booking_id=booking_id,
            booking_status=booking_status,
            creative_due=creative_due,
            currency=currency,
            initials=initials,
            issue=issue,
            issue_on=issue_on,
            list_name=list_name,
            price_cents=price_cents,
            slot_id=slot_id,
            slot_name=slot_name,
            sponsor=sponsor,
        )

        return sponsor_block

