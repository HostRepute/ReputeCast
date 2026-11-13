from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.lead_ads_activity_item_outcome import LeadAdsActivityItemOutcome
from ..models.lead_ads_activity_item_source import LeadAdsActivityItemSource
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.lead_ads_contact_ref import LeadAdsContactRef





T = TypeVar("T", bound="LeadAdsActivityItem")



@_attrs_define
class LeadAdsActivityItem:
    """ 
        Attributes:
            at (datetime.datetime): When ReputeCast recorded it
            form_id (str):
            form_name (str):
            hidden (bool):
            id (str):
            lead_at (datetime.datetime): When the person sent the form
            outcome (LeadAdsActivityItemOutcome):
            page (str):
            source (LeadAdsActivityItemSource):
            contact (LeadAdsContactRef | Unset):
     """

    at: datetime.datetime
    form_id: str
    form_name: str
    hidden: bool
    id: str
    lead_at: datetime.datetime
    outcome: LeadAdsActivityItemOutcome
    page: str
    source: LeadAdsActivityItemSource
    contact: LeadAdsContactRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.lead_ads_contact_ref import LeadAdsContactRef # noqa: PLC0415
        at = self.at.isoformat()

        form_id = self.form_id

        form_name = self.form_name

        hidden = self.hidden

        id = self.id

        lead_at = self.lead_at.isoformat()

        outcome = self.outcome.value

        page = self.page

        source = self.source.value

        contact: dict[str, Any] | Unset = UNSET
        if not isinstance(self.contact, Unset):
            contact = self.contact.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "formId": form_id,
            "formName": form_name,
            "hidden": hidden,
            "id": id,
            "leadAt": lead_at,
            "outcome": outcome,
            "page": page,
            "source": source,
        })
        if contact is not UNSET:
            field_dict["contact"] = contact

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.lead_ads_contact_ref import LeadAdsContactRef # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        form_id = d.pop("formId")

        form_name = d.pop("formName")

        hidden = d.pop("hidden")

        id = d.pop("id")

        lead_at = datetime.datetime.fromisoformat(d.pop("leadAt"))




        outcome = LeadAdsActivityItemOutcome(d.pop("outcome"))




        page = d.pop("page")

        source = LeadAdsActivityItemSource(d.pop("source"))




        _contact = d.pop("contact", UNSET)
        contact: LeadAdsContactRef | Unset
        if isinstance(_contact,  Unset):
            contact = UNSET
        else:
            contact = LeadAdsContactRef.from_dict(_contact)




        lead_ads_activity_item = cls(
            at=at,
            form_id=form_id,
            form_name=form_name,
            hidden=hidden,
            id=id,
            lead_at=lead_at,
            outcome=outcome,
            page=page,
            source=source,
            contact=contact,
        )

        return lead_ads_activity_item

