from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestCheck")



@_attrs_define
class DigestCheck:
    """ 
        Attributes:
            campaign_id (str):
            run_id (str):
            verdict (str):
     """

    campaign_id: str
    run_id: str
    verdict: str





    def to_dict(self) -> dict[str, Any]:
        campaign_id = self.campaign_id

        run_id = self.run_id

        verdict = self.verdict


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "runId": run_id,
            "verdict": verdict,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        run_id = d.pop("runId")

        verdict = d.pop("verdict")

        digest_check = cls(
            campaign_id=campaign_id,
            run_id=run_id,
            verdict=verdict,
        )

        return digest_check

