from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.lead_ads_form import LeadAdsForm





T = TypeVar("T", bound="LeadAdsPage")



@_attrs_define
class LeadAdsPage:
    """ 
        Attributes:
            forms (list[LeadAdsForm]):
            id (str):
            name (str):  Example: Acme Inc.
            page_id (str): Meta's id of the Page
            subscribed_at (datetime.datetime | Unset):
     """

    forms: list[LeadAdsForm]
    id: str
    name: str
    page_id: str
    subscribed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.lead_ads_form import LeadAdsForm # noqa: PLC0415
        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        id = self.id

        name = self.name

        page_id = self.page_id

        subscribed_at: str | Unset = UNSET
        if not isinstance(self.subscribed_at, Unset):
            subscribed_at = self.subscribed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "forms": forms,
            "id": id,
            "name": name,
            "pageId": page_id,
        })
        if subscribed_at is not UNSET:
            field_dict["subscribedAt"] = subscribed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.lead_ads_form import LeadAdsForm # noqa: PLC0415
        d = dict(src_dict)
        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = LeadAdsForm.from_dict(forms_item_data)



            forms.append(forms_item)


        id = d.pop("id")

        name = d.pop("name")

        page_id = d.pop("pageId")

        _subscribed_at = d.pop("subscribedAt", UNSET)
        subscribed_at: datetime.datetime | Unset
        if isinstance(_subscribed_at,  Unset):
            subscribed_at = UNSET
        else:
            subscribed_at = datetime.datetime.fromisoformat(_subscribed_at)




        lead_ads_page = cls(
            forms=forms,
            id=id,
            name=name,
            page_id=page_id,
            subscribed_at=subscribed_at,
        )

        return lead_ads_page

