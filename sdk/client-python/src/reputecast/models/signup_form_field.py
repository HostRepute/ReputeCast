from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_field_type import SignupFormFieldType
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_choice_option import SignupFormChoiceOption





T = TypeVar("T", bound="SignupFormField")



@_attrs_define
class SignupFormField:
    """ 
        Attributes:
            key (str): email, first_name, last_name, company, a custom field key, or a choice question's own key
            label (str): Shown above the input; empty uses the field's name (a choice question needs its question)
            required (bool):
            multiple (bool | Unset): A choice question that takes several answers
            options (list[SignupFormChoiceOption] | Unset): A choice question's answers (2 to 12)
            type_ (SignupFormFieldType | Unset):
     """

    key: str
    label: str
    required: bool
    multiple: bool | Unset = UNSET
    options: list[SignupFormChoiceOption] | Unset = UNSET
    type_: SignupFormFieldType | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_choice_option import SignupFormChoiceOption # noqa: PLC0415
        key = self.key

        label = self.label

        required = self.required

        multiple = self.multiple

        options: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = []
            for options_item_data in self.options:
                options_item = options_item_data.to_dict()
                options.append(options_item)



        type_: str | Unset = UNSET
        if not isinstance(self.type_, Unset):
            type_ = self.type_.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "required": required,
        })
        if multiple is not UNSET:
            field_dict["multiple"] = multiple
        if options is not UNSET:
            field_dict["options"] = options
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_choice_option import SignupFormChoiceOption # noqa: PLC0415
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        required = d.pop("required")

        multiple = d.pop("multiple", UNSET)

        _options = d.pop("options", UNSET)
        options: list[SignupFormChoiceOption] | Unset = UNSET
        if _options is not UNSET:
            options = []
            for options_item_data in _options:
                options_item = SignupFormChoiceOption.from_dict(options_item_data)



                options.append(options_item)


        _type_ = d.pop("type", UNSET)
        type_: SignupFormFieldType | Unset
        if isinstance(_type_,  Unset):
            type_ = UNSET
        else:
            type_ = SignupFormFieldType(_type_)




        signup_form_field = cls(
            key=key,
            label=label,
            required=required,
            multiple=multiple,
            options=options,
            type_=type_,
        )

        return signup_form_field

