from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VerificationCapabilities")



@_attrs_define
class VerificationCapabilities:
    """ 
        Attributes:
            bulk_lists (bool):
            catch_all (bool):
            credit_balance (bool):
            data_region (bool):
            real_time (bool):
     """

    bulk_lists: bool
    catch_all: bool
    credit_balance: bool
    data_region: bool
    real_time: bool





    def to_dict(self) -> dict[str, Any]:
        bulk_lists = self.bulk_lists

        catch_all = self.catch_all

        credit_balance = self.credit_balance

        data_region = self.data_region

        real_time = self.real_time


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bulkLists": bulk_lists,
            "catchAll": catch_all,
            "creditBalance": credit_balance,
            "dataRegion": data_region,
            "realTime": real_time,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bulk_lists = d.pop("bulkLists")

        catch_all = d.pop("catchAll")

        credit_balance = d.pop("creditBalance")

        data_region = d.pop("dataRegion")

        real_time = d.pop("realTime")

        verification_capabilities = cls(
            bulk_lists=bulk_lists,
            catch_all=catch_all,
            credit_balance=credit_balance,
            data_region=data_region,
            real_time=real_time,
        )

        return verification_capabilities

