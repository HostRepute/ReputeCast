from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_variant_input_key import SignupFormVariantInputKey
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_variant_content import SignupFormVariantContent





T = TypeVar("T", bound="SignupFormVariantInput")



@_attrs_define
class SignupFormVariantInput:
    """ 
        Attributes:
            key (SignupFormVariantInputKey):
            weight (int): Percent of visitors; the variants add up to 100
            content (SignupFormVariantContent | Unset):
     """

    key: SignupFormVariantInputKey
    weight: int
    content: SignupFormVariantContent | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_variant_content import SignupFormVariantContent # noqa: PLC0415
        key = self.key.value

        weight = self.weight

        content: dict[str, Any] | Unset = UNSET
        if not isinstance(self.content, Unset):
            content = self.content.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "weight": weight,
        })
        if content is not UNSET:
            field_dict["content"] = content

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_variant_content import SignupFormVariantContent # noqa: PLC0415
        d = dict(src_dict)
        key = SignupFormVariantInputKey(d.pop("key"))




        weight = d.pop("weight")

        _content = d.pop("content", UNSET)
        content: SignupFormVariantContent | Unset
        if isinstance(_content,  Unset):
            content = UNSET
        else:
            content = SignupFormVariantContent.from_dict(_content)




        signup_form_variant_input = cls(
            key=key,
            weight=weight,
            content=content,
        )

        return signup_form_variant_input

