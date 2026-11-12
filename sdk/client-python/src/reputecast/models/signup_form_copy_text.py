from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_copy_text_fields import SignupFormCopyTextFields
  from ..models.signup_form_copy_text_options import SignupFormCopyTextOptions
  from ..models.signup_form_copy_text_steps import SignupFormCopyTextSteps





T = TypeVar("T", bound="SignupFormCopyText")



@_attrs_define
class SignupFormCopyText:
    """ 
        Attributes:
            after_submit (str): What the form says after a signup
            button (str):
            consent_text (str):
            description (str):
            fields (SignupFormCopyTextFields):
            heading (str):
            options (SignupFormCopyTextOptions | Unset):
            steps (SignupFormCopyTextSteps | Unset):
     """

    after_submit: str
    button: str
    consent_text: str
    description: str
    fields: SignupFormCopyTextFields
    heading: str
    options: SignupFormCopyTextOptions | Unset = UNSET
    steps: SignupFormCopyTextSteps | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_copy_text_fields import SignupFormCopyTextFields # noqa: PLC0415
        from ..models.signup_form_copy_text_options import SignupFormCopyTextOptions # noqa: PLC0415
        from ..models.signup_form_copy_text_steps import SignupFormCopyTextSteps # noqa: PLC0415
        after_submit = self.after_submit

        button = self.button

        consent_text = self.consent_text

        description = self.description

        fields = self.fields.to_dict()

        heading = self.heading

        options: dict[str, Any] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = self.options.to_dict()

        steps: dict[str, Any] | Unset = UNSET
        if not isinstance(self.steps, Unset):
            steps = self.steps.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "afterSubmit": after_submit,
            "button": button,
            "consentText": consent_text,
            "description": description,
            "fields": fields,
            "heading": heading,
        })
        if options is not UNSET:
            field_dict["options"] = options
        if steps is not UNSET:
            field_dict["steps"] = steps

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_copy_text_fields import SignupFormCopyTextFields # noqa: PLC0415
        from ..models.signup_form_copy_text_options import SignupFormCopyTextOptions # noqa: PLC0415
        from ..models.signup_form_copy_text_steps import SignupFormCopyTextSteps # noqa: PLC0415
        d = dict(src_dict)
        after_submit = d.pop("afterSubmit")

        button = d.pop("button")

        consent_text = d.pop("consentText")

        description = d.pop("description")

        fields = SignupFormCopyTextFields.from_dict(d.pop("fields"))




        heading = d.pop("heading")

        _options = d.pop("options", UNSET)
        options: SignupFormCopyTextOptions | Unset
        if isinstance(_options,  Unset):
            options = UNSET
        else:
            options = SignupFormCopyTextOptions.from_dict(_options)




        _steps = d.pop("steps", UNSET)
        steps: SignupFormCopyTextSteps | Unset
        if isinstance(_steps,  Unset):
            steps = UNSET
        else:
            steps = SignupFormCopyTextSteps.from_dict(_steps)




        signup_form_copy_text = cls(
            after_submit=after_submit,
            button=button,
            consent_text=consent_text,
            description=description,
            fields=fields,
            heading=heading,
            options=options,
            steps=steps,
        )

        return signup_form_copy_text

