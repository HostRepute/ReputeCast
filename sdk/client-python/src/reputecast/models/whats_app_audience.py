from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.whats_app_audience_countries import WhatsAppAudienceCountries
  from ..models.whats_app_list_consent import WhatsAppListConsent





T = TypeVar("T", bound="WhatsAppAudience")



@_attrs_define
class WhatsAppAudience:
    """ 
        Attributes:
            consent (int):
            countries (WhatsAppAudienceCountries):
            final (int):
            lists (list[WhatsAppListConsent]):
            suppressed (int):
            without (int):
     """

    consent: int
    countries: WhatsAppAudienceCountries
    final: int
    lists: list[WhatsAppListConsent]
    suppressed: int
    without: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_audience_countries import WhatsAppAudienceCountries # noqa: PLC0415
        from ..models.whats_app_list_consent import WhatsAppListConsent # noqa: PLC0415
        consent = self.consent

        countries = self.countries.to_dict()

        final = self.final

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        suppressed = self.suppressed

        without = self.without


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "consent": consent,
            "countries": countries,
            "final": final,
            "lists": lists,
            "suppressed": suppressed,
            "without": without,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_audience_countries import WhatsAppAudienceCountries # noqa: PLC0415
        from ..models.whats_app_list_consent import WhatsAppListConsent # noqa: PLC0415
        d = dict(src_dict)
        consent = d.pop("consent")

        countries = WhatsAppAudienceCountries.from_dict(d.pop("countries"))




        final = d.pop("final")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = WhatsAppListConsent.from_dict(lists_item_data)



            lists.append(lists_item)


        suppressed = d.pop("suppressed")

        without = d.pop("without")

        whats_app_audience = cls(
            consent=consent,
            countries=countries,
            final=final,
            lists=lists,
            suppressed=suppressed,
            without=without,
        )

        return whats_app_audience

