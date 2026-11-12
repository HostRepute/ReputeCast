from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_whats_app_step_mode import AutomationWhatsAppStepMode
from ..models.automation_whats_app_step_outside import AutomationWhatsAppStepOutside
from ..models.automation_whats_app_step_outside_window import AutomationWhatsAppStepOutsideWindow
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_whats_app_var import AutomationWhatsAppVar





T = TypeVar("T", bound="AutomationWhatsAppStep")



@_attrs_define
class AutomationWhatsAppStep:
    """ 
        Attributes:
            consent_list_id (str | Unset):
            fallback_template (str | Unset):
            language (str | Unset):
            mode (AutomationWhatsAppStepMode | Unset):
            outside (AutomationWhatsAppStepOutside | Unset):
            outside_window (AutomationWhatsAppStepOutsideWindow | Unset):
            template (str | Unset):
            text (str | Unset):
            variables (list[AutomationWhatsAppVar] | Unset):
     """

    consent_list_id: str | Unset = UNSET
    fallback_template: str | Unset = UNSET
    language: str | Unset = UNSET
    mode: AutomationWhatsAppStepMode | Unset = UNSET
    outside: AutomationWhatsAppStepOutside | Unset = UNSET
    outside_window: AutomationWhatsAppStepOutsideWindow | Unset = UNSET
    template: str | Unset = UNSET
    text: str | Unset = UNSET
    variables: list[AutomationWhatsAppVar] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_whats_app_var import AutomationWhatsAppVar # noqa: PLC0415
        consent_list_id = self.consent_list_id

        fallback_template = self.fallback_template

        language = self.language

        mode: str | Unset = UNSET
        if not isinstance(self.mode, Unset):
            mode = self.mode.value


        outside: str | Unset = UNSET
        if not isinstance(self.outside, Unset):
            outside = self.outside.value


        outside_window: str | Unset = UNSET
        if not isinstance(self.outside_window, Unset):
            outside_window = self.outside_window.value


        template = self.template

        text = self.text

        variables: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.variables, Unset):
            variables = []
            for variables_item_data in self.variables:
                variables_item = variables_item_data.to_dict()
                variables.append(variables_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if consent_list_id is not UNSET:
            field_dict["consentListId"] = consent_list_id
        if fallback_template is not UNSET:
            field_dict["fallbackTemplate"] = fallback_template
        if language is not UNSET:
            field_dict["language"] = language
        if mode is not UNSET:
            field_dict["mode"] = mode
        if outside is not UNSET:
            field_dict["outside"] = outside
        if outside_window is not UNSET:
            field_dict["outsideWindow"] = outside_window
        if template is not UNSET:
            field_dict["template"] = template
        if text is not UNSET:
            field_dict["text"] = text
        if variables is not UNSET:
            field_dict["variables"] = variables

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_whats_app_var import AutomationWhatsAppVar # noqa: PLC0415
        d = dict(src_dict)
        consent_list_id = d.pop("consentListId", UNSET)

        fallback_template = d.pop("fallbackTemplate", UNSET)

        language = d.pop("language", UNSET)

        _mode = d.pop("mode", UNSET)
        mode: AutomationWhatsAppStepMode | Unset
        if isinstance(_mode,  Unset):
            mode = UNSET
        else:
            mode = AutomationWhatsAppStepMode(_mode)




        _outside = d.pop("outside", UNSET)
        outside: AutomationWhatsAppStepOutside | Unset
        if isinstance(_outside,  Unset):
            outside = UNSET
        else:
            outside = AutomationWhatsAppStepOutside(_outside)




        _outside_window = d.pop("outsideWindow", UNSET)
        outside_window: AutomationWhatsAppStepOutsideWindow | Unset
        if isinstance(_outside_window,  Unset):
            outside_window = UNSET
        else:
            outside_window = AutomationWhatsAppStepOutsideWindow(_outside_window)




        template = d.pop("template", UNSET)

        text = d.pop("text", UNSET)

        _variables = d.pop("variables", UNSET)
        variables: list[AutomationWhatsAppVar] | Unset = UNSET
        if _variables is not UNSET:
            variables = []
            for variables_item_data in _variables:
                variables_item = AutomationWhatsAppVar.from_dict(variables_item_data)



                variables.append(variables_item)


        automation_whats_app_step = cls(
            consent_list_id=consent_list_id,
            fallback_template=fallback_template,
            language=language,
            mode=mode,
            outside=outside,
            outside_window=outside_window,
            template=template,
            text=text,
            variables=variables,
        )

        return automation_whats_app_step

