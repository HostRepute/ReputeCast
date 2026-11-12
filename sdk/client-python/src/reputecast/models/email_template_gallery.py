from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.email_template_last_campaign import EmailTemplateLastCampaign
  from ..models.email_template_summary import EmailTemplateSummary





T = TypeVar("T", bound="EmailTemplateGallery")



@_attrs_define
class EmailTemplateGallery:
    """ 
        Attributes:
            sections (list[EmailTemplateSummary]):
            starters (list[EmailTemplateSummary]):
            templates (list[EmailTemplateSummary]):
            last_campaign (EmailTemplateLastCampaign | Unset):
     """

    sections: list[EmailTemplateSummary]
    starters: list[EmailTemplateSummary]
    templates: list[EmailTemplateSummary]
    last_campaign: EmailTemplateLastCampaign | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.email_template_last_campaign import EmailTemplateLastCampaign # noqa: PLC0415
        from ..models.email_template_summary import EmailTemplateSummary # noqa: PLC0415
        sections = []
        for sections_item_data in self.sections:
            sections_item = sections_item_data.to_dict()
            sections.append(sections_item)



        starters = []
        for starters_item_data in self.starters:
            starters_item = starters_item_data.to_dict()
            starters.append(starters_item)



        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        last_campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_campaign, Unset):
            last_campaign = self.last_campaign.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "sections": sections,
            "starters": starters,
            "templates": templates,
        })
        if last_campaign is not UNSET:
            field_dict["lastCampaign"] = last_campaign

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.email_template_last_campaign import EmailTemplateLastCampaign # noqa: PLC0415
        from ..models.email_template_summary import EmailTemplateSummary # noqa: PLC0415
        d = dict(src_dict)
        sections = []
        _sections = d.pop("sections")
        for sections_item_data in (_sections):
            sections_item = EmailTemplateSummary.from_dict(sections_item_data)



            sections.append(sections_item)


        starters = []
        _starters = d.pop("starters")
        for starters_item_data in (_starters):
            starters_item = EmailTemplateSummary.from_dict(starters_item_data)



            starters.append(starters_item)


        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = EmailTemplateSummary.from_dict(templates_item_data)



            templates.append(templates_item)


        _last_campaign = d.pop("lastCampaign", UNSET)
        last_campaign: EmailTemplateLastCampaign | Unset
        if isinstance(_last_campaign,  Unset):
            last_campaign = UNSET
        else:
            last_campaign = EmailTemplateLastCampaign.from_dict(_last_campaign)




        email_template_gallery = cls(
            sections=sections,
            starters=starters,
            templates=templates,
            last_campaign=last_campaign,
        )

        return email_template_gallery

