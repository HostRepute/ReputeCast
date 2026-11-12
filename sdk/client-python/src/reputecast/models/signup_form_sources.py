from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_sources_referrer import SignupFormSourcesReferrer
from typing import cast






T = TypeVar("T", bound="SignupFormSources")



@_attrs_define
class SignupFormSources:
    """ 
        Attributes:
            referrer (SignupFormSourcesReferrer):
            utm_campaign (list[str]):
            utm_medium (list[str]):
            utm_source (list[str]):
     """

    referrer: SignupFormSourcesReferrer
    utm_campaign: list[str]
    utm_medium: list[str]
    utm_source: list[str]





    def to_dict(self) -> dict[str, Any]:
        referrer = self.referrer.value

        utm_campaign = self.utm_campaign



        utm_medium = self.utm_medium



        utm_source = self.utm_source




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "referrer": referrer,
            "utmCampaign": utm_campaign,
            "utmMedium": utm_medium,
            "utmSource": utm_source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        referrer = SignupFormSourcesReferrer(d.pop("referrer"))




        utm_campaign = cast(list[str], d.pop("utmCampaign"))


        utm_medium = cast(list[str], d.pop("utmMedium"))


        utm_source = cast(list[str], d.pop("utmSource"))


        signup_form_sources = cls(
            referrer=referrer,
            utm_campaign=utm_campaign,
            utm_medium=utm_medium,
            utm_source=utm_source,
        )

        return signup_form_sources

