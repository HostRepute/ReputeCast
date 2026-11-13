from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.lead_ads_checkbox import LeadAdsCheckbox
  from ..models.lead_ads_field_option import LeadAdsFieldOption
  from ..models.lead_ads_form import LeadAdsForm
  from ..models.lead_ads_list_option import LeadAdsListOption
  from ..models.lead_ads_question import LeadAdsQuestion





T = TypeVar("T", bound="LeadAdsFormDetail")



@_attrs_define
class LeadAdsFormDetail:
    """ 
        Attributes:
            checkboxes (list[LeadAdsCheckbox]):
            disclaimer (str): The custom disclaimer's text
            fields (list[LeadAdsFieldOption]):
            form (LeadAdsForm):
            list_ids (list[str]):
            lists (list[LeadAdsListOption]):
            questions (list[LeadAdsQuestion]):
            consent_key (str | Unset):
            consent_text (str | Unset): Single opt-in: the checkbox's text the leads agreed to
     """

    checkboxes: list[LeadAdsCheckbox]
    disclaimer: str
    fields: list[LeadAdsFieldOption]
    form: LeadAdsForm
    list_ids: list[str]
    lists: list[LeadAdsListOption]
    questions: list[LeadAdsQuestion]
    consent_key: str | Unset = UNSET
    consent_text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.lead_ads_checkbox import LeadAdsCheckbox # noqa: PLC0415
        from ..models.lead_ads_field_option import LeadAdsFieldOption # noqa: PLC0415
        from ..models.lead_ads_form import LeadAdsForm # noqa: PLC0415
        from ..models.lead_ads_list_option import LeadAdsListOption # noqa: PLC0415
        from ..models.lead_ads_question import LeadAdsQuestion # noqa: PLC0415
        checkboxes = []
        for checkboxes_item_data in self.checkboxes:
            checkboxes_item = checkboxes_item_data.to_dict()
            checkboxes.append(checkboxes_item)



        disclaimer = self.disclaimer

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        form = self.form.to_dict()

        list_ids = self.list_ids



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        questions = []
        for questions_item_data in self.questions:
            questions_item = questions_item_data.to_dict()
            questions.append(questions_item)



        consent_key = self.consent_key

        consent_text = self.consent_text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkboxes": checkboxes,
            "disclaimer": disclaimer,
            "fields": fields,
            "form": form,
            "listIds": list_ids,
            "lists": lists,
            "questions": questions,
        })
        if consent_key is not UNSET:
            field_dict["consentKey"] = consent_key
        if consent_text is not UNSET:
            field_dict["consentText"] = consent_text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.lead_ads_checkbox import LeadAdsCheckbox # noqa: PLC0415
        from ..models.lead_ads_field_option import LeadAdsFieldOption # noqa: PLC0415
        from ..models.lead_ads_form import LeadAdsForm # noqa: PLC0415
        from ..models.lead_ads_list_option import LeadAdsListOption # noqa: PLC0415
        from ..models.lead_ads_question import LeadAdsQuestion # noqa: PLC0415
        d = dict(src_dict)
        checkboxes = []
        _checkboxes = d.pop("checkboxes")
        for checkboxes_item_data in (_checkboxes):
            checkboxes_item = LeadAdsCheckbox.from_dict(checkboxes_item_data)



            checkboxes.append(checkboxes_item)


        disclaimer = d.pop("disclaimer")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = LeadAdsFieldOption.from_dict(fields_item_data)



            fields.append(fields_item)


        form = LeadAdsForm.from_dict(d.pop("form"))




        list_ids = cast(list[str], d.pop("listIds"))


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = LeadAdsListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        questions = []
        _questions = d.pop("questions")
        for questions_item_data in (_questions):
            questions_item = LeadAdsQuestion.from_dict(questions_item_data)



            questions.append(questions_item)


        consent_key = d.pop("consentKey", UNSET)

        consent_text = d.pop("consentText", UNSET)

        lead_ads_form_detail = cls(
            checkboxes=checkboxes,
            disclaimer=disclaimer,
            fields=fields,
            form=form,
            list_ids=list_ids,
            lists=lists,
            questions=questions,
            consent_key=consent_key,
            consent_text=consent_text,
        )

        return lead_ads_form_detail

