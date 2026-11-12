from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.review_canary_choice import ReviewCanaryChoice
from ..models.review_canary_reasons_item import ReviewCanaryReasonsItem
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.canary_ip import CanaryIP





T = TypeVar("T", bound="ReviewCanary")



@_attrs_define
class ReviewCanary:
    """ 
        Attributes:
            audience (int):
            auto (bool):
            choice (ReviewCanaryChoice):
            never_mailed (int):
            new_ips (list[CanaryIP]):
            planned (bool):
            reasons (list[ReviewCanaryReasonsItem]):
            seeds (int):
            size (int):
            usual (int):
            domain (str | Unset):
     """

    audience: int
    auto: bool
    choice: ReviewCanaryChoice
    never_mailed: int
    new_ips: list[CanaryIP]
    planned: bool
    reasons: list[ReviewCanaryReasonsItem]
    seeds: int
    size: int
    usual: int
    domain: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.canary_ip import CanaryIP # noqa: PLC0415
        audience = self.audience

        auto = self.auto

        choice = self.choice.value

        never_mailed = self.never_mailed

        new_ips = []
        for new_ips_item_data in self.new_ips:
            new_ips_item = new_ips_item_data.to_dict()
            new_ips.append(new_ips_item)



        planned = self.planned

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.value
            reasons.append(reasons_item)



        seeds = self.seeds

        size = self.size

        usual = self.usual

        domain = self.domain


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audience": audience,
            "auto": auto,
            "choice": choice,
            "neverMailed": never_mailed,
            "newIps": new_ips,
            "planned": planned,
            "reasons": reasons,
            "seeds": seeds,
            "size": size,
            "usual": usual,
        })
        if domain is not UNSET:
            field_dict["domain"] = domain

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.canary_ip import CanaryIP # noqa: PLC0415
        d = dict(src_dict)
        audience = d.pop("audience")

        auto = d.pop("auto")

        choice = ReviewCanaryChoice(d.pop("choice"))




        never_mailed = d.pop("neverMailed")

        new_ips = []
        _new_ips = d.pop("newIps")
        for new_ips_item_data in (_new_ips):
            new_ips_item = CanaryIP.from_dict(new_ips_item_data)



            new_ips.append(new_ips_item)


        planned = d.pop("planned")

        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = ReviewCanaryReasonsItem(reasons_item_data)



            reasons.append(reasons_item)


        seeds = d.pop("seeds")

        size = d.pop("size")

        usual = d.pop("usual")

        domain = d.pop("domain", UNSET)

        review_canary = cls(
            audience=audience,
            auto=auto,
            choice=choice,
            never_mailed=never_mailed,
            new_ips=new_ips,
            planned=planned,
            reasons=reasons,
            seeds=seeds,
            size=size,
            usual=usual,
            domain=domain,
        )

        return review_canary

