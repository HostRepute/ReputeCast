from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_variant_input import SignupFormVariantInput





T = TypeVar("T", bound="SignupFormABInput")



@_attrs_define
class SignupFormABInput:
    """ 
        Attributes:
            auto_end (bool): End the test on its own, keeping the winner, once one is clear
            confidence (float):
            min_views (int): Views each variant needs before a winner can be called
            variants (list[SignupFormVariantInput]):
     """

    auto_end: bool
    confidence: float
    min_views: int
    variants: list[SignupFormVariantInput]





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_variant_input import SignupFormVariantInput # noqa: PLC0415
        auto_end = self.auto_end

        confidence = self.confidence

        min_views = self.min_views

        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "autoEnd": auto_end,
            "confidence": confidence,
            "minViews": min_views,
            "variants": variants,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_variant_input import SignupFormVariantInput # noqa: PLC0415
        d = dict(src_dict)
        auto_end = d.pop("autoEnd")

        confidence = d.pop("confidence")

        min_views = d.pop("minViews")

        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = SignupFormVariantInput.from_dict(variants_item_data)



            variants.append(variants_item)


        signup_form_ab_input = cls(
            auto_end=auto_end,
            confidence=confidence,
            min_views=min_views,
            variants=variants,
        )

        return signup_form_ab_input

