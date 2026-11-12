from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.content_check_unknown_domain_trouble import ContentCheckUnknownDomainTrouble






T = TypeVar("T", bound="ContentCheckUnknownDomain")



@_attrs_define
class ContentCheckUnknownDomain:
    """ 
        Attributes:
            domain (str):
            trouble (ContentCheckUnknownDomainTrouble):
     """

    domain: str
    trouble: ContentCheckUnknownDomainTrouble





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        trouble = self.trouble.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "trouble": trouble,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        trouble = ContentCheckUnknownDomainTrouble(d.pop("trouble"))




        content_check_unknown_domain = cls(
            domain=domain,
            trouble=trouble,
        )

        return content_check_unknown_domain

