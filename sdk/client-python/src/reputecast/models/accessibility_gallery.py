from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.accessibility_template_result import AccessibilityTemplateResult





T = TypeVar("T", bound="AccessibilityGallery")



@_attrs_define
class AccessibilityGallery:
    """ 
        Attributes:
            language (str):
            templates (list[AccessibilityTemplateResult]):
            last_campaign (AccessibilityTemplateResult | Unset):
     """

    language: str
    templates: list[AccessibilityTemplateResult]
    last_campaign: AccessibilityTemplateResult | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.accessibility_template_result import AccessibilityTemplateResult # noqa: PLC0415
        language = self.language

        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        last_campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_campaign, Unset):
            last_campaign = self.last_campaign.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "language": language,
            "templates": templates,
        })
        if last_campaign is not UNSET:
            field_dict["lastCampaign"] = last_campaign

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.accessibility_template_result import AccessibilityTemplateResult # noqa: PLC0415
        d = dict(src_dict)
        language = d.pop("language")

        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = AccessibilityTemplateResult.from_dict(templates_item_data)



            templates.append(templates_item)


        _last_campaign = d.pop("lastCampaign", UNSET)
        last_campaign: AccessibilityTemplateResult | Unset
        if isinstance(_last_campaign,  Unset):
            last_campaign = UNSET
        else:
            last_campaign = AccessibilityTemplateResult.from_dict(_last_campaign)




        accessibility_gallery = cls(
            language=language,
            templates=templates,
            last_campaign=last_campaign,
        )

        return accessibility_gallery

