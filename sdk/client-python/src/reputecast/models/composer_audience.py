from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_audience_cadence import ComposerAudienceCadence
from ..models.composer_audience_paid import ComposerAudiencePaid
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ComposerAudience")



@_attrs_define
class ComposerAudience:
    """ 
        Attributes:
            exclude_lists (list[str]):
            exclude_segments (list[str]):
            lists (list[str]):
            segments (list[str]):
            cadence (ComposerAudienceCadence | Unset):
            paid (ComposerAudiencePaid | Unset):
            paid_tier (str | Unset):
     """

    exclude_lists: list[str]
    exclude_segments: list[str]
    lists: list[str]
    segments: list[str]
    cadence: ComposerAudienceCadence | Unset = UNSET
    paid: ComposerAudiencePaid | Unset = UNSET
    paid_tier: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        exclude_lists = self.exclude_lists



        exclude_segments = self.exclude_segments



        lists = self.lists



        segments = self.segments



        cadence: str | Unset = UNSET
        if not isinstance(self.cadence, Unset):
            cadence = self.cadence.value


        paid: str | Unset = UNSET
        if not isinstance(self.paid, Unset):
            paid = self.paid.value


        paid_tier = self.paid_tier


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "excludeLists": exclude_lists,
            "excludeSegments": exclude_segments,
            "lists": lists,
            "segments": segments,
        })
        if cadence is not UNSET:
            field_dict["cadence"] = cadence
        if paid is not UNSET:
            field_dict["paid"] = paid
        if paid_tier is not UNSET:
            field_dict["paidTier"] = paid_tier

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        exclude_lists = cast(list[str], d.pop("excludeLists"))


        exclude_segments = cast(list[str], d.pop("excludeSegments"))


        lists = cast(list[str], d.pop("lists"))


        segments = cast(list[str], d.pop("segments"))


        _cadence = d.pop("cadence", UNSET)
        cadence: ComposerAudienceCadence | Unset
        if isinstance(_cadence,  Unset):
            cadence = UNSET
        else:
            cadence = ComposerAudienceCadence(_cadence)




        _paid = d.pop("paid", UNSET)
        paid: ComposerAudiencePaid | Unset
        if isinstance(_paid,  Unset):
            paid = UNSET
        else:
            paid = ComposerAudiencePaid(_paid)




        paid_tier = d.pop("paidTier", UNSET)

        composer_audience = cls(
            exclude_lists=exclude_lists,
            exclude_segments=exclude_segments,
            lists=lists,
            segments=segments,
            cadence=cadence,
            paid=paid,
            paid_tier=paid_tier,
        )

        return composer_audience

