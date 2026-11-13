from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_ab_variant_key import SignupFormABVariantKey
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_variant_content import SignupFormVariantContent





T = TypeVar("T", bound="SignupFormABVariant")



@_attrs_define
class SignupFormABVariant:
    """ 
        Attributes:
            confirmed (int):
            key (SignupFormABVariantKey):
            rate (float):
            submissions (int):
            views (int):
            weight (int):
            content (SignupFormVariantContent | Unset):
     """

    confirmed: int
    key: SignupFormABVariantKey
    rate: float
    submissions: int
    views: int
    weight: int
    content: SignupFormVariantContent | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_variant_content import SignupFormVariantContent # noqa: PLC0415
        confirmed = self.confirmed

        key = self.key.value

        rate = self.rate

        submissions = self.submissions

        views = self.views

        weight = self.weight

        content: dict[str, Any] | Unset = UNSET
        if not isinstance(self.content, Unset):
            content = self.content.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmed": confirmed,
            "key": key,
            "rate": rate,
            "submissions": submissions,
            "views": views,
            "weight": weight,
        })
        if content is not UNSET:
            field_dict["content"] = content

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_variant_content import SignupFormVariantContent # noqa: PLC0415
        d = dict(src_dict)
        confirmed = d.pop("confirmed")

        key = SignupFormABVariantKey(d.pop("key"))




        rate = d.pop("rate")

        submissions = d.pop("submissions")

        views = d.pop("views")

        weight = d.pop("weight")

        _content = d.pop("content", UNSET)
        content: SignupFormVariantContent | Unset
        if isinstance(_content,  Unset):
            content = UNSET
        else:
            content = SignupFormVariantContent.from_dict(_content)




        signup_form_ab_variant = cls(
            confirmed=confirmed,
            key=key,
            rate=rate,
            submissions=submissions,
            views=views,
            weight=weight,
            content=content,
        )

        return signup_form_ab_variant

