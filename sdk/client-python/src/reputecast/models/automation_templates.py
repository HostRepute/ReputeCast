from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_starter_template import AutomationStarterTemplate





T = TypeVar("T", bound="AutomationTemplates")



@_attrs_define
class AutomationTemplates:
    """ 
        Attributes:
            templates (list[AutomationStarterTemplate]):
     """

    templates: list[AutomationStarterTemplate]





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_starter_template import AutomationStarterTemplate # noqa: PLC0415
        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "templates": templates,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_starter_template import AutomationStarterTemplate # noqa: PLC0415
        d = dict(src_dict)
        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = AutomationStarterTemplate.from_dict(templates_item_data)



            templates.append(templates_item)


        automation_templates = cls(
            templates=templates,
        )

        return automation_templates

