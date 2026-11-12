from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_step_estimate_input_mode import WhatsAppStepEstimateInputMode
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_whats_app_var import AutomationWhatsAppVar





T = TypeVar("T", bound="WhatsAppStepEstimateInput")



@_attrs_define
class WhatsAppStepEstimateInput:
    """ 
        Attributes:
            consent_list_id (str | Unset):
            fallback_template (str | Unset):
            language (str | Unset):
            mode (WhatsAppStepEstimateInputMode | Unset):
            template (str | Unset):
            variables (list[AutomationWhatsAppVar] | Unset):
     """

    consent_list_id: str | Unset = UNSET
    fallback_template: str | Unset = UNSET
    language: str | Unset = UNSET
    mode: WhatsAppStepEstimateInputMode | Unset = UNSET
    template: str | Unset = UNSET
    variables: list[AutomationWhatsAppVar] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_whats_app_var import AutomationWhatsAppVar # noqa: PLC0415
        consent_list_id = self.consent_list_id

        fallback_template = self.fallback_template

        language = self.language

        mode: str | Unset = UNSET
        if not isinstance(self.mode, Unset):
            mode = self.mode.value


        template = self.template

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
        if template is not UNSET:
            field_dict["template"] = template
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
        mode: WhatsAppStepEstimateInputMode | Unset
        if isinstance(_mode,  Unset):
            mode = UNSET
        else:
            mode = WhatsAppStepEstimateInputMode(_mode)




        template = d.pop("template", UNSET)

        _variables = d.pop("variables", UNSET)
        variables: list[AutomationWhatsAppVar] | Unset = UNSET
        if _variables is not UNSET:
            variables = []
            for variables_item_data in _variables:
                variables_item = AutomationWhatsAppVar.from_dict(variables_item_data)



                variables.append(variables_item)


        whats_app_step_estimate_input = cls(
            consent_list_id=consent_list_id,
            fallback_template=fallback_template,
            language=language,
            mode=mode,
            template=template,
            variables=variables,
        )

        return whats_app_step_estimate_input

