from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.lead_ads_form_consent_mode import LeadAdsFormConsentMode
from ..models.lead_ads_form_status import LeadAdsFormStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.lead_ads_import import LeadAdsImport
  from ..models.lead_ads_list_ref import LeadAdsListRef





T = TypeVar("T", bound="LeadAdsForm")



@_attrs_define
class LeadAdsForm:
    """ 
        Attributes:
            consent_mode (LeadAdsFormConsentMode):
            form_id (str): Meta's id of the form
            id (str):
            import_ (LeadAdsImport):
            leads30d (int):
            lists (list[LeadAdsListRef]):
            mapped (bool):
            meta_status (str):
            name (str):  Example: Spring webinar signup.
            page (str):  Example: Acme Inc.
            page_id (str):
            questions (int):
            status (LeadAdsFormStatus):
            tags (list[str]):
            waiting (int):
            last_lead_at (datetime.datetime | Unset):
     """

    consent_mode: LeadAdsFormConsentMode
    form_id: str
    id: str
    import_: LeadAdsImport
    leads30d: int
    lists: list[LeadAdsListRef]
    mapped: bool
    meta_status: str
    name: str
    page: str
    page_id: str
    questions: int
    status: LeadAdsFormStatus
    tags: list[str]
    waiting: int
    last_lead_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.lead_ads_import import LeadAdsImport # noqa: PLC0415
        from ..models.lead_ads_list_ref import LeadAdsListRef # noqa: PLC0415
        consent_mode = self.consent_mode.value

        form_id = self.form_id

        id = self.id

        import_ = self.import_.to_dict()

        leads30d = self.leads30d

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        mapped = self.mapped

        meta_status = self.meta_status

        name = self.name

        page = self.page

        page_id = self.page_id

        questions = self.questions

        status = self.status.value

        tags = self.tags



        waiting = self.waiting

        last_lead_at: str | Unset = UNSET
        if not isinstance(self.last_lead_at, Unset):
            last_lead_at = self.last_lead_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "consentMode": consent_mode,
            "formId": form_id,
            "id": id,
            "import": import_,
            "leads30d": leads30d,
            "lists": lists,
            "mapped": mapped,
            "metaStatus": meta_status,
            "name": name,
            "page": page,
            "pageId": page_id,
            "questions": questions,
            "status": status,
            "tags": tags,
            "waiting": waiting,
        })
        if last_lead_at is not UNSET:
            field_dict["lastLeadAt"] = last_lead_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.lead_ads_import import LeadAdsImport # noqa: PLC0415
        from ..models.lead_ads_list_ref import LeadAdsListRef # noqa: PLC0415
        d = dict(src_dict)
        consent_mode = LeadAdsFormConsentMode(d.pop("consentMode"))




        form_id = d.pop("formId")

        id = d.pop("id")

        import_ = LeadAdsImport.from_dict(d.pop("import"))




        leads30d = d.pop("leads30d")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = LeadAdsListRef.from_dict(lists_item_data)



            lists.append(lists_item)


        mapped = d.pop("mapped")

        meta_status = d.pop("metaStatus")

        name = d.pop("name")

        page = d.pop("page")

        page_id = d.pop("pageId")

        questions = d.pop("questions")

        status = LeadAdsFormStatus(d.pop("status"))




        tags = cast(list[str], d.pop("tags"))


        waiting = d.pop("waiting")

        _last_lead_at = d.pop("lastLeadAt", UNSET)
        last_lead_at: datetime.datetime | Unset
        if isinstance(_last_lead_at,  Unset):
            last_lead_at = UNSET
        else:
            last_lead_at = datetime.datetime.fromisoformat(_last_lead_at)




        lead_ads_form = cls(
            consent_mode=consent_mode,
            form_id=form_id,
            id=id,
            import_=import_,
            leads30d=leads30d,
            lists=lists,
            mapped=mapped,
            meta_status=meta_status,
            name=name,
            page=page,
            page_id=page_id,
            questions=questions,
            status=status,
            tags=tags,
            waiting=waiting,
            last_lead_at=last_lead_at,
        )

        return lead_ads_form

