from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_lead_facts_problem import IntegrationLeadFactsProblem
from ..types import UNSET, Unset






T = TypeVar("T", bound="IntegrationLeadFacts")



@_attrs_define
class IntegrationLeadFacts:
    """ 
        Attributes:
            leads30d (int):
            name (str):
            pages (int):
            polling (bool):
            problem (IntegrationLeadFactsProblem | Unset):
     """

    leads30d: int
    name: str
    pages: int
    polling: bool
    problem: IntegrationLeadFactsProblem | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        leads30d = self.leads30d

        name = self.name

        pages = self.pages

        polling = self.polling

        problem: str | Unset = UNSET
        if not isinstance(self.problem, Unset):
            problem = self.problem.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "leads30d": leads30d,
            "name": name,
            "pages": pages,
            "polling": polling,
        })
        if problem is not UNSET:
            field_dict["problem"] = problem

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        leads30d = d.pop("leads30d")

        name = d.pop("name")

        pages = d.pop("pages")

        polling = d.pop("polling")

        _problem = d.pop("problem", UNSET)
        problem: IntegrationLeadFactsProblem | Unset
        if isinstance(_problem,  Unset):
            problem = UNSET
        else:
            problem = IntegrationLeadFactsProblem(_problem)




        integration_lead_facts = cls(
            leads30d=leads30d,
            name=name,
            pages=pages,
            polling=polling,
            problem=problem,
        )

        return integration_lead_facts

