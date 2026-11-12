from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_price_list_kind import ConnectPriceListKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.connect_price_plan import ConnectPricePlan





T = TypeVar("T", bound="ConnectPriceList")



@_attrs_define
class ConnectPriceList:
    """ 
        Attributes:
            kind (ConnectPriceListKind):
            data_per_mb (float | Unset):
            dedicated_ip (float | Unset):
            dedicated_plans (list[ConnectPricePlan] | Unset):
            from_ (float | Unset):
            per_thousand (float | Unset):
            plans (list[ConnectPricePlan] | Unset):
     """

    kind: ConnectPriceListKind
    data_per_mb: float | Unset = UNSET
    dedicated_ip: float | Unset = UNSET
    dedicated_plans: list[ConnectPricePlan] | Unset = UNSET
    from_: float | Unset = UNSET
    per_thousand: float | Unset = UNSET
    plans: list[ConnectPricePlan] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.connect_price_plan import ConnectPricePlan # noqa: PLC0415
        kind = self.kind.value

        data_per_mb = self.data_per_mb

        dedicated_ip = self.dedicated_ip

        dedicated_plans: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.dedicated_plans, Unset):
            dedicated_plans = []
            for dedicated_plans_item_data in self.dedicated_plans:
                dedicated_plans_item = dedicated_plans_item_data.to_dict()
                dedicated_plans.append(dedicated_plans_item)



        from_ = self.from_

        per_thousand = self.per_thousand

        plans: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.plans, Unset):
            plans = []
            for plans_item_data in self.plans:
                plans_item = plans_item_data.to_dict()
                plans.append(plans_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if data_per_mb is not UNSET:
            field_dict["dataPerMb"] = data_per_mb
        if dedicated_ip is not UNSET:
            field_dict["dedicatedIp"] = dedicated_ip
        if dedicated_plans is not UNSET:
            field_dict["dedicatedPlans"] = dedicated_plans
        if from_ is not UNSET:
            field_dict["from"] = from_
        if per_thousand is not UNSET:
            field_dict["perThousand"] = per_thousand
        if plans is not UNSET:
            field_dict["plans"] = plans

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.connect_price_plan import ConnectPricePlan # noqa: PLC0415
        d = dict(src_dict)
        kind = ConnectPriceListKind(d.pop("kind"))




        data_per_mb = d.pop("dataPerMb", UNSET)

        dedicated_ip = d.pop("dedicatedIp", UNSET)

        _dedicated_plans = d.pop("dedicatedPlans", UNSET)
        dedicated_plans: list[ConnectPricePlan] | Unset = UNSET
        if _dedicated_plans is not UNSET:
            dedicated_plans = []
            for dedicated_plans_item_data in _dedicated_plans:
                dedicated_plans_item = ConnectPricePlan.from_dict(dedicated_plans_item_data)



                dedicated_plans.append(dedicated_plans_item)


        from_ = d.pop("from", UNSET)

        per_thousand = d.pop("perThousand", UNSET)

        _plans = d.pop("plans", UNSET)
        plans: list[ConnectPricePlan] | Unset = UNSET
        if _plans is not UNSET:
            plans = []
            for plans_item_data in _plans:
                plans_item = ConnectPricePlan.from_dict(plans_item_data)



                plans.append(plans_item)


        connect_price_list = cls(
            kind=kind,
            data_per_mb=data_per_mb,
            dedicated_ip=dedicated_ip,
            dedicated_plans=dedicated_plans,
            from_=from_,
            per_thousand=per_thousand,
            plans=plans,
        )

        return connect_price_list

