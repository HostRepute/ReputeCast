from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_audience_input_cadence import ComposerAudienceInputCadence
from ..models.composer_audience_input_paid import ComposerAudienceInputPaid
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ComposerAudienceInput")



@_attrs_define
class ComposerAudienceInput:
    """ 
        Attributes:
            cadence (ComposerAudienceInputCadence | Unset): regular (default): readers who get every issue; announcement:
                everyone, whatever rhythm they chose
            exclude_lists (list[str] | Unset):
            exclude_segments (list[str] | Unset):
            lists (list[str] | Unset):
            paid (ComposerAudienceInputPaid | Unset): paid: paid members only; free: free subscribers only; empty: everyone
                on the lists
            paid_tier (str | Unset): With paid: the lowest tier (the tier and above); empty for any paid tier
            segments (list[str] | Unset):
     """

    cadence: ComposerAudienceInputCadence | Unset = UNSET
    exclude_lists: list[str] | Unset = UNSET
    exclude_segments: list[str] | Unset = UNSET
    lists: list[str] | Unset = UNSET
    paid: ComposerAudienceInputPaid | Unset = UNSET
    paid_tier: str | Unset = UNSET
    segments: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        cadence: str | Unset = UNSET
        if not isinstance(self.cadence, Unset):
            cadence = self.cadence.value


        exclude_lists: list[str] | Unset = UNSET
        if not isinstance(self.exclude_lists, Unset):
            exclude_lists = self.exclude_lists



        exclude_segments: list[str] | Unset = UNSET
        if not isinstance(self.exclude_segments, Unset):
            exclude_segments = self.exclude_segments



        lists: list[str] | Unset = UNSET
        if not isinstance(self.lists, Unset):
            lists = self.lists



        paid: str | Unset = UNSET
        if not isinstance(self.paid, Unset):
            paid = self.paid.value


        paid_tier = self.paid_tier

        segments: list[str] | Unset = UNSET
        if not isinstance(self.segments, Unset):
            segments = self.segments




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if cadence is not UNSET:
            field_dict["cadence"] = cadence
        if exclude_lists is not UNSET:
            field_dict["excludeLists"] = exclude_lists
        if exclude_segments is not UNSET:
            field_dict["excludeSegments"] = exclude_segments
        if lists is not UNSET:
            field_dict["lists"] = lists
        if paid is not UNSET:
            field_dict["paid"] = paid
        if paid_tier is not UNSET:
            field_dict["paidTier"] = paid_tier
        if segments is not UNSET:
            field_dict["segments"] = segments

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _cadence = d.pop("cadence", UNSET)
        cadence: ComposerAudienceInputCadence | Unset
        if isinstance(_cadence,  Unset):
            cadence = UNSET
        else:
            cadence = ComposerAudienceInputCadence(_cadence)




        exclude_lists = cast(list[str], d.pop("excludeLists", UNSET))


        exclude_segments = cast(list[str], d.pop("excludeSegments", UNSET))


        lists = cast(list[str], d.pop("lists", UNSET))


        _paid = d.pop("paid", UNSET)
        paid: ComposerAudienceInputPaid | Unset
        if isinstance(_paid,  Unset):
            paid = UNSET
        else:
            paid = ComposerAudienceInputPaid(_paid)




        paid_tier = d.pop("paidTier", UNSET)

        segments = cast(list[str], d.pop("segments", UNSET))


        composer_audience_input = cls(
            cadence=cadence,
            exclude_lists=exclude_lists,
            exclude_segments=exclude_segments,
            lists=lists,
            paid=paid,
            paid_tier=paid_tier,
            segments=segments,
        )

        return composer_audience_input

