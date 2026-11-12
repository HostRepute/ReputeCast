from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.provider_limits_input import ProviderLimitsInput
  from ..models.provider_pricing_input import ProviderPricingInput





T = TypeVar("T", bound="ProviderSettingsInput")



@_attrs_define
class ProviderSettingsInput:
    """ 
        Attributes:
            label (str | Unset):
            limits (ProviderLimitsInput | Unset):
            pricing (ProviderPricingInput | Unset):
     """

    label: str | Unset = UNSET
    limits: ProviderLimitsInput | Unset = UNSET
    pricing: ProviderPricingInput | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_limits_input import ProviderLimitsInput # noqa: PLC0415
        from ..models.provider_pricing_input import ProviderPricingInput # noqa: PLC0415
        label = self.label

        limits: dict[str, Any] | Unset = UNSET
        if not isinstance(self.limits, Unset):
            limits = self.limits.to_dict()

        pricing: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pricing, Unset):
            pricing = self.pricing.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if label is not UNSET:
            field_dict["label"] = label
        if limits is not UNSET:
            field_dict["limits"] = limits
        if pricing is not UNSET:
            field_dict["pricing"] = pricing

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_limits_input import ProviderLimitsInput # noqa: PLC0415
        from ..models.provider_pricing_input import ProviderPricingInput # noqa: PLC0415
        d = dict(src_dict)
        label = d.pop("label", UNSET)

        _limits = d.pop("limits", UNSET)
        limits: ProviderLimitsInput | Unset
        if isinstance(_limits,  Unset):
            limits = UNSET
        else:
            limits = ProviderLimitsInput.from_dict(_limits)




        _pricing = d.pop("pricing", UNSET)
        pricing: ProviderPricingInput | Unset
        if isinstance(_pricing,  Unset):
            pricing = UNSET
        else:
            pricing = ProviderPricingInput.from_dict(_pricing)




        provider_settings_input = cls(
            label=label,
            limits=limits,
            pricing=pricing,
        )

        return provider_settings_input

